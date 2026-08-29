package prometheus

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	api "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type Querier interface {
	Query(ctx context.Context, request QueryRequest) (Result, error)
}

type Client struct {
	api     v1.API
	timeout time.Duration
}

type QueryRequest struct {
	Expr    string
	Start   time.Time
	End     time.Time
	Step    time.Duration
	Instant bool
}

type Result struct {
	ResultType string
	Series     []Series
	Scalar     *Sample
	Text       string
	Warnings   []string
}

type Series struct {
	Labels  map[string]string
	Samples []Sample
}

type Sample struct {
	Timestamp time.Time
	Value     float64
}

func NewClient(address, bearerToken string) (*Client, error) {
	transport := api.DefaultRoundTripper
	if bearerToken != "" {
		transport = bearerRoundTripper{token: bearerToken, next: transport}
	}
	client, err := api.NewClient(api.Config{Address: address, RoundTripper: transport})
	if err != nil {
		return nil, fmt.Errorf("create Prometheus client: %w", err)
	}
	return &Client{api: v1.NewAPI(client), timeout: 30 * time.Second}, nil
}

func (c *Client) Query(ctx context.Context, request QueryRequest) (Result, error) {
	if request.Expr == "" {
		return Result{}, fmt.Errorf("PromQL expression must not be empty")
	}
	if request.End.IsZero() {
		request.End = time.Now()
	}
	if request.Start.IsZero() {
		request.Start = request.End.Add(-6 * time.Hour)
	}
	if request.Step <= 0 {
		request.Step = time.Minute
	}

	queryCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if request.Instant {
		value, warnings, err := c.api.Query(queryCtx, request.Expr, request.End)
		if err != nil {
			return Result{}, fmt.Errorf("execute instant query: %w", err)
		}
		return normalize(value, warnings), nil
	}

	value, warnings, err := c.api.QueryRange(queryCtx, request.Expr, v1.Range{
		Start: request.Start,
		End:   request.End,
		Step:  request.Step,
	})
	if err != nil {
		return Result{}, fmt.Errorf("execute range query: %w", err)
	}
	return normalize(value, warnings), nil
}

type bearerRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (t bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(clone)
}

func normalize(value model.Value, warnings v1.Warnings) Result {
	result := Result{
		ResultType: value.Type().String(),
		Warnings:   append([]string(nil), warnings...),
	}
	switch typed := value.(type) {
	case model.Matrix:
		for _, stream := range typed {
			result.Series = append(result.Series, Series{
				Labels:  labelMap(stream.Metric),
				Samples: samplePairs(stream.Values),
			})
		}
	case model.Vector:
		for _, sample := range typed {
			result.Series = append(result.Series, Series{
				Labels:  labelMap(sample.Metric),
				Samples: []Sample{{Timestamp: time.UnixMilli(int64(sample.Timestamp)), Value: float64(sample.Value)}},
			})
		}
	case *model.Scalar:
		if typed != nil {
			result.Scalar = &Sample{Timestamp: time.UnixMilli(int64(typed.Timestamp)), Value: float64(typed.Value)}
		}
	case *model.String:
		if typed != nil {
			result.Text = typed.Value
		}
	}
	return result
}

func labelMap(labels model.Metric) map[string]string {
	result := make(map[string]string, len(labels))
	for name, value := range labels {
		result[string(name)] = string(value)
	}
	return result
}

func samplePairs(pairs []model.SamplePair) []Sample {
	result := make([]Sample, 0, len(pairs))
	for _, pair := range pairs {
		value := float64(pair.Value)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		result = append(result, Sample{Timestamp: time.UnixMilli(int64(pair.Timestamp)), Value: value})
	}
	return result
}

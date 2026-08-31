package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GrafanaSource struct {
	baseURL *url.URL
	token   string
	client  *http.Client
}

func NewGrafanaSource(rawURL, token string) (*GrafanaSource, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Grafana URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("grafana URL must use http or https")
	}
	return &GrafanaSource{
		baseURL: parsed,
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *GrafanaSource) List(ctx context.Context) ([]DashboardSummary, error) {
	const pageSize = 1000
	var dashboards []DashboardSummary
	for page := 1; ; page++ {
		query := url.Values{"type": {"dash-db"}, "limit": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}}
		var results []searchResult
		if err := s.request(ctx, "/api/search", query, &results); err != nil {
			return nil, fmt.Errorf("list Grafana dashboards: %w", err)
		}
		for _, result := range results {
			id := result.UID
			if id == "" && result.ID != 0 {
				id = strconv.FormatInt(result.ID, 10)
			}
			dashboards = append(dashboards, DashboardSummary{
				ID: id, UID: result.UID, Title: result.Title,
				Tags: result.Tags, URL: result.URL, FolderUID: result.FolderUID,
				Starred: result.IsStarred, Source: "grafana",
			})
		}
		if len(results) < pageSize {
			break
		}
	}
	return dashboards, nil
}

func (s *GrafanaSource) Get(ctx context.Context, id string) (Dashboard, error) {
	var payload json.RawMessage
	if err := s.request(ctx, "/api/dashboards/uid/"+url.PathEscape(id), nil, &payload); err != nil {
		return Dashboard{}, fmt.Errorf("get Grafana dashboard %q: %w", id, err)
	}
	dashboard, err := Parse(payload)
	if err != nil {
		return Dashboard{}, fmt.Errorf("parse Grafana dashboard %q: %w", id, err)
	}
	dashboard.ID = id
	dashboard.Source = "grafana"
	if dashboard.UID == "" {
		dashboard.UID = id
	}
	return dashboard, nil
}

type searchResult struct {
	ID        int64    `json:"id"`
	UID       string   `json:"uid"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	FolderUID string   `json:"folderUid"`
	Tags      []string `json:"tags"`
	IsStarred bool     `json:"isStarred"`
}

func (s *GrafanaSource) request(ctx context.Context, path string, query url.Values, result any) error {
	u := *s.baseURL
	u.Path = strings.TrimRight(s.baseURL.Path, "/") + path
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("grafana returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode Grafana response: %w", err)
	}
	return nil
}

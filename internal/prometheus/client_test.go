package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/common/model"
)

func TestClientQueryRangeNormalizesMatrix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			http.Error(writer, "missing auth", http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
  "status":"success",
  "data":{"resultType":"matrix","result":[{"metric":{"__name__":"up","job":"demo"},"values":[[1700000000,"1"],[1700000060,"2"]]}]},
  "warnings":["slow query"]
}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Query(context.Background(), QueryRequest{
		Expr:  "up",
		Start: time.Unix(1700000000, 0),
		End:   time.Unix(1700000060, 0),
		Step:  time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultType != "matrix" || len(result.Series) != 1 || len(result.Series[0].Samples) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Series[0].Labels["job"] != "demo" || len(result.Warnings) != 1 {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
}

func TestNormalizeScalar(t *testing.T) {
	result := normalize(&model.Scalar{Timestamp: model.Time(1700000000000), Value: 42}, nil)
	if result.Scalar == nil || result.Scalar.Value != 42 {
		t.Fatalf("unexpected scalar result: %+v", result)
	}
}

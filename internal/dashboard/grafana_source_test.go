package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGrafanaSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			http.Error(writer, "missing auth", http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/search":
			_ = json.NewEncoder(writer).Encode([]searchResult{{UID: "apollo", Title: "Apollo"}})
		case "/api/dashboards/uid/apollo":
			_, _ = writer.Write([]byte(`{"dashboard":{"uid":"apollo","title":"Apollo","panels":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	source, err := NewGrafanaSource(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := source.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].UID != "apollo" {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	loaded, err := source.Get(context.Background(), "apollo")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Apollo" || loaded.Source != "grafana" {
		t.Fatalf("unexpected dashboard: %+v", loaded)
	}
}

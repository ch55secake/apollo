package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSourceListAndGet(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "classic.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dashboard.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}

	source, err := NewFileSource(dir)
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := source.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Title != "Apollo Classic" {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	loaded, err := source.Get(context.Background(), summaries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Source != "file" || loaded.ID != summaries[0].ID {
		t.Fatalf("unexpected loaded dashboard: %+v", loaded.DashboardSummary)
	}
}

func TestFileSourceRejectsPathOutsideDirectory(t *testing.T) {
	dir := t.TempDir()
	source, err := NewFileSource(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Get(context.Background(), filepath.Join(dir, "..", "outside.json")); err == nil {
		t.Fatal("expected path traversal to fail")
	}
}

package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileSource struct {
	path string
}

func NewFileSource(path string) (*FileSource, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve dashboard path: %w", err)
	}
	return &FileSource{path: filepath.Clean(absPath)}, nil
}

func (s *FileSource) List(ctx context.Context) ([]DashboardSummary, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return nil, fmt.Errorf("stat dashboard path: %w", err)
	}
	if !info.IsDir() {
		return s.summaryForFile(s.path)
	}
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("read dashboard directory: %w", err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			paths = append(paths, filepath.Join(s.path, entry.Name()))
		}
	}
	sort.Strings(paths)

	summaries := make([]DashboardSummary, 0, len(paths))
	for _, path := range paths {
		items, err := s.summaryForFile(path)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, items...)
	}
	return summaries, nil
}

func (s *FileSource) Get(ctx context.Context, id string) (Dashboard, error) {
	select {
	case <-ctx.Done():
		return Dashboard{}, ctx.Err()
	default:
	}
	path, err := s.resolveID(id)
	if err != nil {
		return Dashboard{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Dashboard{}, fmt.Errorf("read dashboard %q: %w", path, err)
	}
	dashboard, err := Parse(data)
	if err != nil {
		return Dashboard{}, fmt.Errorf("parse dashboard %q: %w", path, err)
	}
	dashboard.ID = path
	dashboard.Source = "file"
	if dashboard.UID == "" {
		dashboard.UID = filepath.Base(path)
	}
	return dashboard, nil
}

func (s *FileSource) summaryForFile(path string) ([]DashboardSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dashboard %q: %w", path, err)
	}
	dashboard, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse dashboard %q: %w", path, err)
	}
	dashboard.ID = path
	dashboard.Source = "file"
	if dashboard.UID == "" {
		dashboard.UID = filepath.Base(path)
	}
	return []DashboardSummary{dashboard.DashboardSummary}, nil
}

func (s *FileSource) resolveID(id string) (string, error) {
	path := id
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.path, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve dashboard ID: %w", err)
	}
	if info, statErr := os.Stat(s.path); statErr == nil && info.IsDir() {
		rel, relErr := filepath.Rel(s.path, path)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("dashboard ID is outside configured directory")
		}
	} else if filepath.Clean(path) != filepath.Clean(s.path) {
		return "", fmt.Errorf("dashboard ID does not match configured file")
	}
	return path, nil
}

package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderBrandUsesFullMarkWhenItFits(t *testing.T) {
	width := lipgloss.Width(fullLogo) + 4
	if rendered := renderBrand(width, 30); !strings.Contains(rendered, "████") {
		t.Fatalf("expected full Apollo mark, got %q", rendered)
	}
}

func TestRenderBrandFallsBackToCompactWordmark(t *testing.T) {
	if rendered := renderBrand(40, 12); !strings.Contains(rendered, "APOLLO") || strings.Contains(rendered, "████") {
		t.Fatalf("expected compact Apollo wordmark, got %q", rendered)
	}
}

func TestTruncatePreservesUnicode(t *testing.T) {
	value := truncate("latency λ metrics", 12)
	if !utf8.ValidString(value) {
		t.Fatalf("truncate returned invalid UTF-8: %q", value)
	}
}

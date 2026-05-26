package ui

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/dpuwork/differ/internal/theme"
)

func TestTokenForeground_Set(t *testing.T) {
	t.Parallel()
	entry := chroma.StyleEntry{Colour: chroma.MustParseColour("#ff0000")}
	got := tokenForeground(entry)
	if got == "" {
		t.Error("expected non-empty foreground for set colour")
	}
}

func TestTokenForeground_Unset(t *testing.T) {
	t.Parallel()
	entry := chroma.StyleEntry{}
	got := tokenForeground(entry)
	if got != "" {
		t.Errorf("expected empty foreground for unset colour, got %q", got)
	}
}

func TestHighlightLine_Empty(t *testing.T) {
	t.Parallel()
	// Empty content should return empty regardless of chroma state
	got := highlightLine("", "test.go", "")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestHighlightLine_GoCode(t *testing.T) {
	t.Parallel()
	_, th := testStyles()
	initChromaStyle(th)

	got := highlightLine("func main() {}", "main.go", "")
	if got == "" {
		t.Error("expected non-empty highlighted output")
	}
}

func TestHighlightLine_BackgroundLightVsDark(t *testing.T) {
	// Reset the background to dark, init, and check
	thDark := theme.Theme{ChromaStyle: "auto", IsDark: true}
	chromaStyle = nil
	initChromaStyle(thDark)
	if chromaStyle.Name != "monokai" {
		t.Errorf("expected monokai style for dark background, got %q", chromaStyle.Name)
	}

	// Change background to light, init, and check
	thLight := theme.Theme{ChromaStyle: "auto", IsDark: false}
	chromaStyle = nil
	initChromaStyle(thLight)
	if chromaStyle.Name != "monokailight" {
		t.Errorf("expected monokailight style for light background, got %q", chromaStyle.Name)
	}
}

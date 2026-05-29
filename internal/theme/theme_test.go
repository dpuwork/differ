package theme

import (
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.Name != "terminal" {
		t.Errorf("expected theme name 'terminal', got %q", th.Name)
	}
}

func TestGetTheme(t *testing.T) {
	th := GetTheme(false)
	if th.Name != "terminal" {
		t.Errorf("expected theme name 'terminal', got %q", th.Name)
	}
}


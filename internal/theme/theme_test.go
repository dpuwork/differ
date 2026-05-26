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
	th := GetTheme("")
	if th.Name != "terminal" {
		t.Errorf("expected theme name 'terminal', got %q", th.Name)
	}
}

func TestIsDarkBackgroundOverrides(t *testing.T) {
	// Test config light override
	if IsDarkBackground("light") {
		t.Error("expected light theme from config option")
	}

	// Test config dark override
	if !IsDarkBackground("dark") {
		t.Error("expected dark theme from config option")
	}
}


package theme

import (
	"os"

	"charm.land/lipgloss/v2"
)

// IsDarkBackground detects if the background is dark based on terminal query.
func IsDarkBackground() bool {
	// Fall back to automatic detection before raw mode starts
	return lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
}

// Theme defines color values for the UI.
type Theme struct {
	Name string

	// Diff colors
	AddedFg   string
	AddedBg   string
	RemovedFg string
	RemovedBg string
	HunkFg    string

	// Line numbers
	LineNumFg        string
	LineNumAddedFg   string
	LineNumRemovedFg string

	// Header bar
	HeaderBg string
	HeaderFg string

	// File list
	SelectedFg  string
	StagedFg    string
	ModifiedFg  string
	AddedFileFg string
	DeletedFg   string
	RenamedFg   string
	UntrackedFg string
	TypeChangedFg string
	UnmergedFg    string
	IgnoredFg     string
	UnknownFg     string

	// Chrome
	BorderFg    string
	StatusBarBg string
	StatusBarFg string
	HelpKeyFg   string
	HelpDescFg  string

	// Accent
	AccentFg string

	// Chroma syntax theme name
	ChromaStyle string
	IsDark      bool
}

// DefaultTheme returns the single theme used by differ.
// It uses ANSI colors to respect terminal themes.
func DefaultTheme() Theme {
	return Theme{
		Name: "terminal",

		// Diff colors - using ANSI colors for FG
		AddedFg:   "2", // Green
		RemovedFg: "1", // Red
		HunkFg:    "6", // Cyan

		LineNumAddedFg:   "2", // Green
		LineNumRemovedFg: "1", // Red

		SelectedFg:  "6", // Cyan
		StagedFg:    "2", // Green
		ModifiedFg:  "3", // Yellow
		AddedFileFg: "2", // Green
		DeletedFg:   "1", // Red
		RenamedFg:   "4", // Blue
		UntrackedFg: "8", // Gray
		TypeChangedFg: "5", // Magenta
		UnmergedFg:    "9", // Bright Red
		IgnoredFg:     "8", // Gray
		UnknownFg:     "7", // Gray

		BorderFg:  "6", // Cyan
		HelpKeyFg: "6", // Cyan

		AccentFg: "6", // Cyan

		// Use "auto" to switch between monokai and monokailight
		ChromaStyle: "auto",
	}
}

// GetTheme returns the theme, with values adjusted for detected background.
func GetTheme(isDark bool) Theme {
	t := DefaultTheme()
	t.IsDark = isDark
	if t.IsDark {
		t.AddedBg = "#233b2a"
		t.RemovedBg = "#3b232e"
		t.LineNumFg = "8"
		t.StatusBarBg = ""
		t.StatusBarFg = "7"
		t.HelpDescFg = "7"
		t.HeaderBg = "8"
		t.HeaderFg = "15"
	} else {
		t.AddedBg = "#e6f5e4"
		t.RemovedBg = "#fde4e8"
		t.LineNumFg = "7"
		t.StatusBarBg = ""
		t.StatusBarFg = "0"
		t.HelpDescFg = "0"
		t.HeaderBg = "7"
		t.HeaderFg = "0"
	}
	return t
}

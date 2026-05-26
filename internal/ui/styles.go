package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/dpuwork/differ/internal/theme"
)

// Styles holds all lipgloss styles derived from a theme.
type Styles struct {
	// File list
	FileItem     lipgloss.Style
	FileSelected lipgloss.Style
	StagedIcon   lipgloss.Style

	// File status colors
	StatusModified  lipgloss.Style
	StatusAdded     lipgloss.Style
	StatusDeleted   lipgloss.Style
	StatusRenamed   lipgloss.Style
	StatusUntracked lipgloss.Style
	StatusTypeChanged lipgloss.Style
	StatusUnmerged    lipgloss.Style
	StatusIgnored     lipgloss.Style
	StatusUnknown     lipgloss.Style

	// Diff
	DiffAdded           lipgloss.Style
	DiffRemoved         lipgloss.Style
	DiffAddedBg         lipgloss.Style // bg-only, for padding highlighted lines
	DiffRemovedBg       lipgloss.Style // bg-only, for padding highlighted lines
	DiffContext         lipgloss.Style
	DiffHunkHeader      lipgloss.Style
	DiffLineNum         lipgloss.Style
	DiffLineNumAdded    lipgloss.Style
	DiffLineNumRemoved  lipgloss.Style

	// Chrome
	HeaderBar   lipgloss.Style
	StatusBar   lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Accent
	Accent lipgloss.Style
	ListHeader lipgloss.Style
}

// NewStyles creates styles from a theme.
func NewStyles(t theme.Theme) Styles {
	s := Styles{
		FileItem: lipgloss.NewStyle(),
		FileSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SelectedFg)).
			Bold(true),
		StagedIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StagedFg)).
			Bold(true),

		StatusModified: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ModifiedFg)),
		StatusAdded: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.AddedFileFg)),
		StatusDeleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.DeletedFg)),
		StatusRenamed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.RenamedFg)),
		StatusUntracked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.UntrackedFg)),
		StatusTypeChanged: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TypeChangedFg)),
		StatusUnmerged: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.UnmergedFg)).
			Bold(true),
		StatusIgnored: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.IgnoredFg)),
		StatusUnknown: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.UnknownFg)),

		DiffAdded:      lipgloss.NewStyle().Foreground(lipgloss.Color(t.AddedFg)),
		DiffRemoved:    lipgloss.NewStyle().Foreground(lipgloss.Color(t.RemovedFg)),
		DiffContext:    lipgloss.NewStyle(),
		DiffHunkHeader: lipgloss.NewStyle().Foreground(lipgloss.Color(t.HunkFg)),
		DiffLineNum:    lipgloss.NewStyle().Foreground(lipgloss.Color(t.LineNumFg)),
		DiffLineNumAdded: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.LineNumAddedFg)),
		DiffLineNumRemoved: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.LineNumRemovedFg)),

		HeaderBar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.HeaderBg)).
			Foreground(lipgloss.Color(t.HeaderFg)).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBar: lipgloss.NewStyle(),
		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.HelpKeyFg)).
			Bold(true),
		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.HelpDescFg)),

		Accent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.AccentFg)),
		ListHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.HelpDescFg)).
			Bold(true).
			PaddingLeft(1),
	}

	if t.AddedBg != "" {
		s.DiffAdded = s.DiffAdded.Background(lipgloss.Color(t.AddedBg))
		s.DiffAddedBg = lipgloss.NewStyle().Background(lipgloss.Color(t.AddedBg))
		s.DiffLineNumAdded = s.DiffLineNumAdded.Background(lipgloss.Color(t.AddedBg))
	}
	if t.RemovedBg != "" {
		s.DiffRemoved = s.DiffRemoved.Background(lipgloss.Color(t.RemovedBg))
		s.DiffRemovedBg = lipgloss.NewStyle().Background(lipgloss.Color(t.RemovedBg))
		s.DiffLineNumRemoved = s.DiffLineNumRemoved.Background(lipgloss.Color(t.RemovedBg))
	}

	return s
}

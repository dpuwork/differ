package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderHelpPopup() string {
	type helpEntry struct {
		key  string
		desc string
	}
	type helpSection struct {
		title   string
		entries []helpEntry
	}

	leftSections := []helpSection{
		{
			title: "General",
			entries: []helpEntry{
				{"?", "toggle help"},
				{"q", "quit"},
				{"s", "toggle split diff"},
				{"w", "toggle text wrap"},
				{"b", "branch picker"},
				{"esc", "back / cancel"},
			},
		},
		{
			title: "Navigation (File List)",
			entries: []helpEntry{
				{"j/k, ↓/↑", "navigate files"},
				{"g/G", "top / bottom"},
				{"enter, l, →", "view diff"},
			},
		},
		{
			title: "Navigation (Diff)",
			entries: []helpEntry{
				{"j/k, ↓/↑", "scroll diff"},
				{"d/u", "half page down/up"},
				{"n/p", "next / previous file"},
				{"h, ←", "back to list"},
			},
		},
	}

	rightSections := []helpSection{
		{
			title: "Actions",
			entries: []helpEntry{
				{"tab", "stage / unstage file"},
				{"x", "discard unstaged changes"},
				{"a", "stage all changes"},
				{"c", "commit staged changes"},
				{"e", "open in editor & quit"},
				{"P", "push to upstream"},
				{"F", "pull from upstream"},
			},
		},
		{
			title: "Branch Picker",
			entries: []helpEntry{
				{"type", "filter branches"},
				{"enter", "switch to branch"},
				{"ctrl+n", "create new branch"},
			},
		},
	}

	renderSection := func(s helpSection) string {
		var b strings.Builder
		b.WriteString("  " + m.styles.HelpKey.Underline(true).Render(s.title))
		b.WriteByte('\n')
		for _, e := range s.entries {
			keyStr := e.key
			keyWidth := lipgloss.Width(keyStr)
			padding := ""
			if keyWidth < 12 {
				padding = strings.Repeat(" ", 12-keyWidth)
			}
			key := m.styles.HelpKey.Render(keyStr) + padding
			desc := e.desc
			fmt.Fprintf(&b, "    %s %s\n", key, desc)
		}
		return b.String()
	}

	var leftColBuilder strings.Builder
	for i, s := range leftSections {
		leftColBuilder.WriteString(renderSection(s))
		if i < len(leftSections)-1 {
			leftColBuilder.WriteByte('\n')
		}
	}

	var rightColBuilder strings.Builder
	for i, s := range rightSections {
		rightColBuilder.WriteString(renderSection(s))
		if i < len(rightSections)-1 {
			rightColBuilder.WriteByte('\n')
		}
	}

	colGap := "    "
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftColBuilder.String(), colGap, rightColBuilder.String())
	content = "\n" + content + "\n"

	pw := 86
	ph := 20
	if pw > m.width-2 {
		pw = m.width - 2
	}
	if ph > m.height-2 {
		ph = m.height - 2
	}

	title := "Help"
	if m.version != "" {
		title = fmt.Sprintf("Help (%s)", m.version)
	}

	return m.renderCard(title, content, true, pw, ph)
}

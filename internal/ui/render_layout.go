package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/dpuwork/differ/internal/git"
	"github.com/dpuwork/differ/internal/theme"
)

// View composition and all rendering helpers.

func (m Model) View() tea.View {
	if m.width == 0 || !m.ready {
		return tea.NewView("")
	}
	if m.width < minWidth || m.height < minHeight {
		return tea.NewView(fmt.Sprintf("Terminal too small (%dx%d). Minimum: %dx%d", m.width, m.height, minWidth, minHeight))
	}

	var content string
	if m.mode == modeHelp {
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderHelpPopup())
	} else {
		contentH := m.contentHeight()
		var fileContent string
		if m.mode == modeBranchPicker {
			fileContent = m.renderBranchList(contentH)
		} else {
			fileContent = m.renderFileList(contentH)
		}
		fileCard := m.renderCard(m.fileCardTitle(), fileContent, m.mode == modeFileList || m.mode == modeBranchPicker, fileListWidth, contentH)
		diffCard := m.renderCard(m.diffCardTitle(), m.viewport.View(), m.mode == modeDiff, m.diffWidth(), contentH)
		main := lipgloss.JoinHorizontal(lipgloss.Top, fileCard, " ", diffCard)
		statusBar := m.renderStatusBar()

		if m.mode == modeCommit {
			content = lipgloss.JoinVertical(lipgloss.Left, main, statusBar, m.renderCommitBar())
		} else if m.mode == modeBranchPicker && m.branchCreating {
			content = lipgloss.JoinVertical(lipgloss.Left, main, statusBar, m.renderBranchCreateBar())
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) renderCard(title, content string, focused bool, w, h int) string {
	return renderCard(m.theme, title, content, focused, w, h)
}

func renderCard(t theme.Theme, title, content string, focused bool, w, h int) string {
	borderColor := lipgloss.Color(t.HelpDescFg) // Gray for inactive
	if focused {
		borderColor = lipgloss.Color(t.AccentFg) // Cyan for active
	}
	bs := lipgloss.NewStyle().Foreground(borderColor)

	tl, tr := "╭", "╮"
	bl, br := "╰", "╯"
	ls, rs := "│", "│"
	ts, bts := "─", "─"

	titleStr := ""
	if title != "" {
		titleStr = " " + title + " "
	}
	topFill := w - lipgloss.Width(titleStr) - 1
	if topFill < 0 {
		topFill = 0
	}
	top := bs.Render(tl + ts + titleStr + strings.Repeat(ts, topFill) + tr)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.HelpDescFg))

	lines := strings.Split(content, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	var rows []string
	for i := 0; i < h; i++ {
		line := lines[i]
		if !focused && line != "" {
			line = dimStyle.Render(line)
		}
		pad := w - lipgloss.Width(line)
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		rows = append(rows, bs.Render(ls)+line+bs.Render(rs))
	}
	bottom := bs.Render(bl + strings.Repeat(bts, w) + br)
	return lipgloss.JoinVertical(lipgloss.Left, top, strings.Join(rows, "\n"), bottom)
}

func (m Model) fileCardTitle() string {
	if m.mode == modeBranchPicker {
		return "Branches"
	}
	title := m.repo.BranchName()
	if m.ref != "" {
		title += " ref:" + m.ref
	} else if m.stagedOnly {
		title += " staged"
	}
	return title
}

func (m Model) diffCardTitle() string {
	if len(m.files) == 0 || m.cursor >= len(m.files) {
		return ""
	}
	f := m.files[m.cursor]
	name := f.change.Path
	if f.change.Staged {
		name += " [staged]"
	}
	return name
}

func (m Model) renderFileList(height int) string {
	var b strings.Builder
	renderedLines := 0

	stagedCount := 0
	for _, f := range m.files {
		if f.change.Staged {
			stagedCount++
		}
	}

	if stagedCount > 0 {
		b.WriteString(m.styles.ListHeader.Render("STAGED"))
		b.WriteByte('\n')
		renderedLines++

		for i := 0; i < stagedCount; i++ {
			if renderedLines >= height {
				break
			}
			b.WriteString(m.renderFileItem(m.files[i], i == m.cursor))
			renderedLines++
			if renderedLines < height && (i < stagedCount-1 || len(m.files) > stagedCount) {
				b.WriteByte('\n')
			}
		}
	}

	if len(m.files) > stagedCount && renderedLines < height {
		header := m.styles.ListHeader
		if stagedCount > 0 {
			header = header.MarginTop(1)
		}
		b.WriteString(header.Render("CHANGES"))
		b.WriteByte('\n')
		renderedLines++

		for i := stagedCount; i < len(m.files); i++ {
			if renderedLines >= height {
				break
			}
			b.WriteString(m.renderFileItem(m.files[i], i == m.cursor))
			renderedLines++
			if renderedLines < height && i < len(m.files)-1 {
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}

func (m Model) renderFileItem(f fileItem, selected bool) string {
	status := string(f.change.Status)
	addedStr := fmt.Sprintf("+%d", f.change.AddedLines)
	deletedStr := fmt.Sprintf("-%d", f.change.DeletedLines)
	statsWidth := len(addedStr) + 1 + len(deletedStr)

	name := filepath.Base(f.change.Path)
	if f.change.OldPath != "" {
		name = filepath.Base(f.change.OldPath) + " → " + filepath.Base(f.change.Path)
	}

	statusWidth := lipgloss.Width(status)
	nameMaxW := fileListWidth - 1 - statusWidth - 1 - statsWidth - 2
	if nameMaxW < 1 {
		nameMaxW = 1
	}
	name = TruncatePath(name, nameMaxW)
	nameWidth := lipgloss.Width(name)

	gap := fileListWidth - 1 - statusWidth - 1 - nameWidth - statsWidth - 1
	if gap < 1 {
		gap = 1
	}

	var statusStyled, nameStyled, gapStr, addedStyled, deletedStyled, leftSpace, midSpace, rightSpace string

	if selected {
		statusStyled = m.styles.FileSelected.Render(status)
		nameStyled = m.styles.FileSelected.Render(name)
		gapStr = m.styles.FileSelected.Render(strings.Repeat(" ", gap))
		addedStyled = m.styles.FileStatsAdded.Bold(true).Render(addedStr)
		deletedStyled = m.styles.FileStatsDeleted.Bold(true).Render(deletedStr)
		leftSpace = m.styles.FileSelected.Render(" ")
		midSpace = m.styles.FileSelected.Render(" ")
		rightSpace = m.styles.FileSelected.Render(" ")
	} else {
		statusStyled = m.styleStatus(status, f.change.Status)
		nameStyled = m.styles.FileItem.Render(name)
		gapStr = strings.Repeat(" ", gap)
		addedStyled = m.styles.FileStatsAdded.Render(addedStr)
		deletedStyled = m.styles.FileStatsDeleted.Render(deletedStr)
		leftSpace = " "
		midSpace = " "
		rightSpace = " "
	}

	statsStyled := addedStyled + " " + deletedStyled

	return leftSpace + statusStyled + midSpace + nameStyled + gapStr + statsStyled + rightSpace
}

func (m Model) renderBranchList(height int) string {
	var b strings.Builder
	b.WriteString(m.renderBranchFilterBar())
	b.WriteByte('\n')
	list := m.activeBranches()
	itemH := height - 1
	if len(list) == 0 {
		b.WriteString(m.styles.FileItem.Width(fileListWidth).Render(m.styles.HelpDesc.Render("  no matches")))
		return b.String()
	}
	end := m.branchOffset + itemH
	if end > len(list) {
		end = len(list)
	}
	for i := m.branchOffset; i < end; i++ {
		b.WriteString(m.renderBranchItem(list[i], i == m.branchCursor, list[i] == m.currentBranch))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderBranchFilterBar() string {
	list := m.activeBranches()
	countStyled := m.styles.HelpDesc.Render(fmt.Sprintf("%d/%d", len(list), len(m.branches)))
	input := m.branchFilter.View()
	gap := fileListWidth - lipgloss.Width(input) - lipgloss.Width(countStyled) - 1
	if gap < 0 {
		gap = 0
	}
	return lipgloss.NewStyle().Width(fileListWidth).Render(input + strings.Repeat(" ", gap) + countStyled)
}

func (m Model) renderBranchItem(name string, selected, current bool) string {
	prefix := " "
	if current {
		prefix = m.styles.StagedIcon.Render("*")
	}
	line := prefix + TruncatePath(name, fileListWidth-2)
	if selected {
		return m.styles.FileSelected.Width(fileListWidth).MaxHeight(1).Render(line)
	}
	return m.styles.FileItem.Width(fileListWidth).MaxHeight(1).Render(line)
}

func TruncatePath(path string, maxW int) string {
	if lipgloss.Width(path) <= maxW {
		return path
	}
	for lipgloss.Width(path) > maxW-1 && len(path) > 1 {
		path = path[1:]
	}
	return "…" + path
}

func (m Model) styleStatus(icon string, status git.FileStatus) string {
	switch status {
	case git.StatusModified:
		return m.styles.StatusModified.Render(icon)
	case git.StatusAdded:
		return m.styles.StatusAdded.Render(icon)
	case git.StatusDeleted:
		return m.styles.StatusDeleted.Render(icon)
	case git.StatusRenamed:
		return m.styles.StatusRenamed.Render(icon)
	case git.StatusUntracked:
		return m.styles.StatusUntracked.Render(icon)
	case git.StatusTypeChanged:
		return m.styles.StatusTypeChanged.Render(icon)
	case git.StatusUnmerged:
		return m.styles.StatusUnmerged.Render(icon)
	case git.StatusIgnored:
		return m.styles.StatusIgnored.Render(icon)
	case git.StatusUnknown:
		return m.styles.StatusUnknown.Render(icon)
	default:
		return icon
	}
}

func (m Model) renderStatusBar() string {
	stagedCount := 0
	for _, f := range m.files {
		if f.change.Staged {
			stagedCount++
		}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%d staged  %d files", stagedCount, len(m.files)))

	if m.upstream.Upstream != "" {
		parts = append(parts, fmt.Sprintf("↑%d ↓%d", m.upstream.Ahead, m.upstream.Behind))
	}

	var modes []string
	if m.splitDiff {
		modes = append(modes, "split")
	}
	if m.wrapDiff {
		modes = append(modes, "wrap")
	}
	if len(modes) > 0 {
		parts = append(parts, strings.Join(modes, " "))
	}

	if m.statusMsg != "" {
		msg := m.statusMsg
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		parts = append(parts, msg)
	}

	divider := m.styles.HelpDesc.Render(" │ ")
	left := " " + strings.Join(parts, divider)

	help := "(?) help"
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(help) - 1
	if gap < 0 {
		gap = 0
	}

	content := left + strings.Repeat(" ", gap) + help
	return m.styles.StatusBar.Width(m.width).Render(content)
}

func (m Model) renderCommitBar() string {
	prompt := m.styles.HelpKey.Render(" commit: ")
	return lipgloss.NewStyle().Width(m.width).Render(prompt + m.commitInput.View() + "  " + "esc cancel · enter commit")
}

func (m Model) renderBranchCreateBar() string {
	prompt := m.styles.HelpKey.Render(" new branch: ")
	return lipgloss.NewStyle().Width(m.width).Render(prompt + m.branchInput.View() + "  " + "esc cancel · enter create")
}

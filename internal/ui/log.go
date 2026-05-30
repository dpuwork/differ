package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dpuwork/differ/internal/config"
	"github.com/dpuwork/differ/internal/git"
	"github.com/dpuwork/differ/internal/theme"
)

type logMode int

const (
	logModeList logMode = iota
	logModeDiff
	logModeHelp
)

type logLoadedMsg struct {
	commits []git.Commit
}

type logDiffLoadedMsg struct {
	content string
	hash    string
}

// LogModel is the Bubble Tea model for the commit log browser.
type LogModel struct {
	repo     *git.Repo
	cfg      config.Config
	styles   Styles
	theme    theme.Theme
	version  string
	commits  []git.Commit
	cursor   int
	mode     logMode
	prevMode logMode
	viewport viewport.Model
	width    int
	height   int
	ready    bool

	splitDiff bool
	wrapDiff  bool
}

// NewLogModel creates the log browser model.
func NewLogModel(repo *git.Repo, cfg config.Config, styles Styles, t theme.Theme, version string) LogModel {
	return LogModel{
		repo:      repo,
		cfg:       cfg,
		styles:    styles,
		theme:     t,
		version:   version,
		splitDiff: cfg.SplitDiff,
		wrapDiff:  cfg.WrapDiff,
	}
}

func (m LogModel) Init() tea.Cmd {
	repo := m.repo
	return tea.Batch(
		func() tea.Msg {
			commits, _ := repo.Log(100)
			return logLoadedMsg{commits: commits}
		},
		tickThemeCmd(),
	)
}

func (m *LogModel) refreshTheme() {
	m.styles = NewStyles(m.theme)
}

func (m LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		isDark := msg.IsDark()
		if m.theme.IsDark != isDark {
			m.theme = theme.GetTheme(isDark)
			m.refreshTheme()
			if m.mode == logModeDiff && len(m.commits) > 0 {
				return m, m.loadCommitDiff()
			}
			return m, nil
		}
		return m, nil
	case tickThemeMsg:
		return m, tea.Batch(tea.RequestBackgroundColor, tickThemeCmd())
	case tea.WindowSizeMsg:
		m.refreshTheme()
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(viewport.WithWidth(m.width-2), viewport.WithHeight(m.height-3))
		m.ready = true
		if m.mode == logModeDiff {
			return m, m.loadCommitDiff()
		}
	case logLoadedMsg:
		m.commits = msg.commits
	case logDiffLoadedMsg:
		m.viewport.SetContent(msg.content)
		m.viewport.GotoTop()
		m.mode = logModeDiff
	case tea.KeyPressMsg:
		switch m.mode {
		case logModeList:
			return m.updateList(msg)
		case logModeDiff:
			return m.updateDiff(msg)
		case logModeHelp:
			return m.updateHelp(msg)
		}
	}
	return m, nil
}

func (m LogModel) updateHelp(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc", "?":
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

func (m LogModel) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.prevMode = m.mode
		m.mode = logModeHelp
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.commits)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = max(0, len(m.commits)-1)
	case "enter":
		if len(m.commits) > 0 {
			return m, m.loadCommitDiff()
		}
	case "s":
		m.splitDiff = !m.splitDiff
		if m.mode == logModeDiff {
			return m, m.loadCommitDiff()
		}
	case "w":
		m.wrapDiff = !m.wrapDiff
		if m.mode == logModeDiff {
			return m, m.loadCommitDiff()
		}
	}
	return m, nil
}

func (m LogModel) updateDiff(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.prevMode = m.mode
		m.mode = logModeHelp
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = logModeList
		return m, nil
	case "s":
		m.splitDiff = !m.splitDiff
		return m, m.loadCommitDiff()
	case "w":
		m.wrapDiff = !m.wrapDiff
		return m, m.loadCommitDiff()
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m LogModel) loadCommitDiff() tea.Cmd {
	commit := m.commits[m.cursor]
	repo := m.repo
	styles := m.styles
	t := m.theme
	width := m.width

	return func() tea.Msg {
		raw, err := repo.CommitDiff(commit.Hash)
		if err != nil {
			return logDiffLoadedMsg{content: "Error: " + err.Error(), hash: commit.Hash}
		}
		// Guess filename from diff headers for syntax highlighting
		content := renderCommitDiff(raw, styles, t, width, m.splitDiff, m.wrapDiff)
		return logDiffLoadedMsg{content: content, hash: commit.Hash}
	}
}

// renderCommitDiff renders a full commit diff (may contain multiple files).
func renderCommitDiff(raw string, styles Styles, t theme.Theme, width int, split, wrap bool) string {
	initChromaStyle(t)

	var b strings.Builder
	// Split by file boundaries and render each section
	currentFile := ""
	var currentLines []string

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "diff --git") {
			// Flush previous file
			if len(currentLines) > 0 {
				parsed := ParseDiff(strings.Join(currentLines, "\n"))
				if split {
					b.WriteString(RenderSplitDiff(parsed, currentFile, styles, t, width, wrap))
				} else {
					b.WriteString(RenderDiff(parsed, currentFile, styles, t, width, wrap))
				}
			}
			currentFile = extractFilename(line)
			// Add file separator
			b.WriteString(styles.HeaderBar.Width(width).Render(" " + currentFile))
			b.WriteByte('\n')
			currentLines = []string{line}
		} else {
			currentLines = append(currentLines, line)
		}
	}
	// Flush last file
	if len(currentLines) > 0 {
		parsed := ParseDiff(strings.Join(currentLines, "\n"))
		if split {
			b.WriteString(RenderSplitDiff(parsed, currentFile, styles, t, width, wrap))
		} else {
			b.WriteString(RenderDiff(parsed, currentFile, styles, t, width, wrap))
		}
	}
	return b.String()
}

// extractFilename pulls the b/ path from "diff --git a/foo b/foo".
func extractFilename(diffHeader string) string {
	parts := strings.SplitN(diffHeader, " b/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func (m LogModel) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}

	var content string
	if m.mode == logModeHelp {
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, RenderHelpPopup(m.styles, m.theme, m.width, m.height, m.version))
	} else {
		switch m.mode {
		case logModeDiff:
			content = m.viewDiff()
		default:
			content = m.viewList()
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m LogModel) viewList() string {
	contentH := m.height - 3 // card borders + status
	cardW := m.width - 2     // inner width (card adds 2 for borders)

	var b strings.Builder
	renderedLines := 0

	b.WriteString(m.styles.ListHeader.Render("COMMITS"))
	b.WriteByte('\n')
	renderedLines++

	for i, c := range m.commits {
		if renderedLines >= contentH {
			break
		}
		line := m.renderCommitLine(c, i == m.cursor)
		b.WriteString(line)
		renderedLines++
		if renderedLines < contentH && i < len(m.commits)-1 {
			b.WriteByte('\n')
		}
	}

	card := renderCard(m.theme, "Commits", b.String(), true, cardW, contentH)
	return lipgloss.JoinVertical(lipgloss.Left, card, m.renderStatusBar())
}

func (m LogModel) renderCommitLine(c git.Commit, selected bool) string {
	width := m.width - 2
	hash := c.Short
	subject := c.Subject
	date := c.Date

	hashWidth := lipgloss.Width(hash)
	dateWidth := lipgloss.Width(date)

	// available = width - leftSpace(1) - hash - midSpace(1) - rightSpace(1) - date
	availableForSubjectAndGap := width - 1 - hashWidth - 1 - dateWidth - 1
	if availableForSubjectAndGap < 5 {
		availableForSubjectAndGap = 5
	}

	subject = TruncatePath(subject, availableForSubjectAndGap-1)
	subjectWidth := lipgloss.Width(subject)

	gap := availableForSubjectAndGap - subjectWidth
	if gap < 0 {
		gap = 0
	}

	var hashStyled, subjectStyled, dateStyled, gapStr, leftSpace, midSpace, rightSpace string

	if selected {
		hashStyled = m.styles.FileSelected.Render(hash)
		subjectStyled = m.styles.FileSelected.Render(subject)
		dateStyled = m.styles.FileSelected.Render(date)
		gapStr = m.styles.FileSelected.Render(strings.Repeat(" ", gap))
		leftSpace = m.styles.FileSelected.Render(" ")
		midSpace = m.styles.FileSelected.Render(" ")
		rightSpace = m.styles.FileSelected.Render(" ")
	} else {
		hashStyled = m.styles.Accent.Render(hash)
		subjectStyled = m.styles.FileItem.Render(subject)
		dateStyled = m.styles.HelpDesc.Render(date)
		gapStr = strings.Repeat(" ", gap)
		leftSpace = " "
		midSpace = " "
		rightSpace = " "
	}

	return leftSpace + hashStyled + midSpace + subjectStyled + gapStr + dateStyled + rightSpace
}

func (m LogModel) viewDiff() string {
	contentH := m.height - 3
	cardW := m.width - 2

	c := m.commits[m.cursor]
	title := c.Short + " " + c.Subject
	card := renderCard(m.theme, title, m.viewport.View(), true, cardW, contentH)
	return lipgloss.JoinVertical(lipgloss.Left, card, m.renderStatusBar())
}

func (m LogModel) renderStatusBar() string {
	var leftParts []string
	var midParts []string
	
	if m.mode == logModeDiff && len(m.commits) > 0 {
		c := m.commits[m.cursor]
		leftParts = append(leftParts, c.Short)
		// We'll handle the subject separately to make it dynamic
		midParts = append(midParts, c.Author)
	} else {
		leftParts = append(leftParts, fmt.Sprintf("%d commits", len(m.commits)))
	}

	// Status modes (split/wrap)
	var modes []string
	if m.splitDiff {
		modes = append(modes, "split")
	}
	if m.wrapDiff {
		modes = append(modes, "wrap")
	}
	if len(modes) > 0 {
		midParts = append(midParts, strings.Join(modes, " "))
	}

	divider := m.styles.HelpDesc.Render(" │ ")
	
	// Right side static parts
	var helpParts []string
	if m.mode == logModeDiff {
		helpParts = append(helpParts, m.styles.HelpKey.Render("esc")+" back")
	} else {
		helpParts = append(helpParts, m.styles.HelpKey.Render("enter")+" view")
	}
	helpParts = append(helpParts, m.styles.HelpKey.Render("q")+" quit")

	rightShortcuts := strings.Join(helpParts, "  ") + " "
	helpToggle := m.styles.HelpDesc.Render(" │ ") + "(?) help"
	rightSide := rightShortcuts + helpToggle
	rightWidth := lipgloss.Width(rightSide)

	// Calculate how much space we have for the subject
	leftFixed := " " + strings.Join(leftParts, divider)
	if len(midParts) > 0 {
		leftFixed += divider + strings.Join(midParts, divider)
	}
	leftWidth := lipgloss.Width(leftFixed)

	// Final assembly
	var content string
	if m.mode == logModeDiff && len(m.commits) > 0 {
		c := m.commits[m.cursor]
		// Reserve some padding for the gaps
		available := m.width - leftWidth - rightWidth - 4
		subject := c.Subject
		if available > 10 {
			if lipgloss.Width(subject) > available {
				subject = TruncatePath(subject, available)
			}
		} else if available <= 0 {
			subject = "" // Terminal is too narrow
		}

		// Re-construct with the dynamic subject
		statusLeft := " " + leftParts[0] + divider + subject
		if len(midParts) > 0 {
			statusLeft += divider + strings.Join(midParts, divider)
		}
		
		gap := m.width - lipgloss.Width(statusLeft) - rightWidth - 1
		if gap < 0 { gap = 0 }
		content = statusLeft + strings.Repeat(" ", gap) + rightSide
	} else {
		gap := m.width - leftWidth - rightWidth - 1
		if gap < 0 { gap = 0 }
		content = leftFixed + strings.Repeat(" ", gap) + rightSide
	}

	return m.styles.StatusBar.Width(m.width).Render(content)
}

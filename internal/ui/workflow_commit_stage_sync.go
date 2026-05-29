package ui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/dpuwork/differ/internal/config"
)

const maxDisplayFileSize = 50 * 1024 * 1024 // 50 MB

func isLargeOrBinary(repoDir, filename string) (bool, string) {
	fullPath := filepath.Join(repoDir, filename)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, ""
	}

	if info.Size() > maxDisplayFileSize {
		return true, "file too large to display"
	}

	// Read first 512 bytes to check for binary
	f, err := os.Open(fullPath)
	if err != nil {
		return false, ""
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n > 0 {
		if strings.Contains(string(buf[:n]), "\x00") {
			return true, "binary file not displayed"
		}
	}

	return false, ""
}

// Commit, staging, polling, sync, and async command workflows.

func (m Model) updateCommitMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeFileList
		m.commitInput.Reset()
		return m, nil
	case "enter":
		message := m.commitInput.Value()
		if strings.TrimSpace(message) == "" {
			var clearCmd tea.Cmd
			m, clearCmd = m.setStatus("empty commit message", true)
			return m, clearCmd
		}
		return m, m.commitCmd(message)
	}
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m Model) toggleStage() (tea.Model, tea.Cmd) {
	if m.stagedOnly || m.ref != "" || len(m.files) == 0 {
		return m, nil
	}
	f := m.files[m.cursor]
	repo := m.repo
	path := f.change.Path
	return m, func() tea.Msg {
		if f.change.Staged {
			_ = repo.UnstageFile(path)
		} else {
			_ = repo.StageFile(path)
		}
		return m.buildRefreshedFiles()
	}
}

func (m Model) stageAll() (tea.Model, tea.Cmd) {
	if m.stagedOnly || m.ref != "" {
		return m, nil
	}
	repo := m.repo
	return m, func() tea.Msg {
		_ = repo.StageAll()
		return m.buildRefreshedFiles()
	}
}

func (m Model) enterCommitMode() (tea.Model, tea.Cmd) {
	if m.ref != "" {
		return m, nil
	}
	hasStaged := false
	for _, f := range m.files {
		if f.change.Staged {
			hasStaged = true
			break
		}
	}
	if !hasStaged {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("no staged files", true)
		return m, clearCmd
	}
	m.mode = modeCommit
	m.commitInput.Focus()
	return m, textinput.Blink
}

func (m Model) fetchUpstreamStatusCmd() tea.Cmd {
	repo := m.repo
	return func() tea.Msg { return upstreamStatusMsg{info: repo.UpstreamStatus()} }
}

func (m Model) pushCmd() tea.Cmd {
	repo := m.repo
	return func() tea.Msg { return pushDoneMsg{err: repo.Push()} }
}

func (m Model) pushSetUpstreamCmd() tea.Cmd {
	repo := m.repo
	branch := m.currentBranch
	if branch == "" {
		branch = repo.BranchName()
	}
	return func() tea.Msg { return pushDoneMsg{err: repo.PushSetUpstream("origin", branch)} }
}

func (m Model) pullCmd() tea.Cmd {
	repo := m.repo
	return func() tea.Msg { return pullDoneMsg{err: repo.Pull()} }
}

type fetchRemoteDoneMsg struct{ err error }

func (m Model) fetchRemoteCmd() tea.Cmd {
	if m.upstream.Upstream == "" {
		return nil
	}
	repo := m.repo
	return func() tea.Msg {
		return fetchRemoteDoneMsg{err: repo.Fetch()}
	}
}

const fetchInterval = 60 * time.Second

func tickFetchCmd() tea.Cmd {
	return tea.Tick(fetchInterval, func(t time.Time) tea.Msg { return tickFetchMsg(t) })
}

type tickFetchMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	m.refreshTheme()
	if m.mode == modeCommit || m.mode == modeBranchPicker {
		return m, tickCmd()
	}
	return m, tea.Batch(m.refreshFilesCmd(), m.fetchUpstreamStatusCmd(), tickCmd())
}

func (m Model) handleTickFetch() (tea.Model, tea.Cmd) {
	if m.mode == modeCommit || m.mode == modeBranchPicker {
		return m, tickFetchCmd()
	}
	return m, tea.Batch(m.fetchRemoteCmd(), tickFetchCmd())
}

func (m Model) handleFetchRemoteDone(msg fetchRemoteDoneMsg) (tea.Model, tea.Cmd) {
	var clearCmd tea.Cmd
	if msg.err != nil {
		m, clearCmd = m.setStatus("fetch failed", true)
	}
	// After fetching from remote, update the upstream status counts
	return m, tea.Batch(m.fetchUpstreamStatusCmd(), clearCmd)
}

func (m Model) handleUpstreamStatus(msg upstreamStatusMsg) (tea.Model, tea.Cmd) {
	m.upstream = msg.info
	if m.upstream.Upstream != "" && !m.initialFetchDone {
		m.initialFetchDone = true
		return m, m.fetchRemoteCmd()
	}
	return m, nil
}

func (m Model) handlePushDone(msg pushDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("push failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	var clearCmd tea.Cmd
	m, clearCmd = m.setStatus("pushed!", true)
	return m, tea.Batch(m.fetchUpstreamStatusCmd(), clearCmd)
}

func (m Model) handlePullDone(msg pullDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("pull failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	var clearCmd tea.Cmd
	m, clearCmd = m.setStatus("pulled!", true)
	return m, tea.Batch(m.refreshFilesCmd(), m.fetchUpstreamStatusCmd(), clearCmd)
}

func (m Model) loadDiffCmd(resetScroll bool) tea.Cmd {
	if len(m.files) == 0 || m.width <= 0 {
		return nil
	}
	idx := m.cursor
	f := m.files[idx]
	repo := m.repo
	styles := m.styles
	t := m.theme
	staged := f.change.Staged
	ref := m.ref
	diffW := m.diffWidth()
	filename := f.change.Path
	splitMode := m.splitDiff && diffW >= minSplitWidth
	return func() tea.Msg {
		var content string
		
		isSkip, skipMsg := isLargeOrBinary(repo.Dir(), filename)
		if isSkip {
			// Center the message or just show it
			content = styles.DiffHunkHeader.Render(skipMsg)
			return diffLoadedMsg{content: content, index: idx, resetScroll: resetScroll}
		}

		if f.untracked {
			raw, err := repo.ReadFileContent(filename)
			if err != nil {
				content = styles.DiffHunkHeader.Render("Error: " + err.Error())
			} else if splitMode {
				content = RenderNewFileSplit(raw, filename, styles, t, diffW, m.wrapDiff)
			} else {
				content = RenderNewFile(raw, filename, styles, t, diffW, m.wrapDiff)
			}
		} else {
			raw, err := repo.DiffFile(filename, staged, ref)
			if err != nil {
				content = styles.DiffHunkHeader.Render("Error: " + err.Error())
			} else {
				parsed := ParseDiff(raw)
				if splitMode {
					content = RenderSplitDiff(parsed, filename, styles, t, diffW, m.wrapDiff)
				} else {
					content = RenderDiff(parsed, filename, styles, t, diffW, m.wrapDiff)
				}
			}
		}
		return diffLoadedMsg{content: content, index: idx, resetScroll: resetScroll}
	}
}

func (m Model) refreshFilesCmd() tea.Cmd {
	repo := m.repo
	stagedOnly := m.stagedOnly
	ref := m.ref
	return func() tea.Msg {
		files, _ := repo.ChangedFiles(stagedOnly, ref)
		var untracked []string
		if !stagedOnly && ref == "" {
			untracked, _ = repo.UntrackedFiles()
		}
		return filesRefreshedMsg{files: buildFileItems(repo, files, untracked)}
	}
}

func (m Model) buildRefreshedFiles() filesRefreshedMsg {
	files, _ := m.repo.ChangedFiles(m.stagedOnly, m.ref)
	var untracked []string
	if !m.stagedOnly && m.ref == "" {
		untracked, _ = m.repo.UntrackedFiles()
	}
	return filesRefreshedMsg{files: buildFileItems(m.repo, files, untracked)}
}

func (m Model) saveConfigCmd() tea.Cmd {
	cfg := m.cfg
	split := m.splitDiff
	wrap := m.wrapDiff
	return func() tea.Msg {
		cfg.SplitDiff = split
		cfg.WrapDiff = wrap
		return savePrefDoneMsg{err: config.Save(cfg)}
	}
}

func (m Model) commitCmd(message string) tea.Cmd {
	repo := m.repo
	return func() tea.Msg { return commitDoneMsg{err: repo.Commit(message)} }
}

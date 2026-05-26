package ui

import tea "charm.land/bubbletea/v2"

// File-list mode input handling and file navigation actions.

func (m Model) updateFileListMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	if msg.String() == "P" {
		if m.pushConfirm {
			m.pushConfirm = false
			m.statusMsg = "pushing..."
			if m.upstream.Upstream == "" {
				return m, m.pushSetUpstreamCmd()
			}
			return m, m.pushCmd()
		}
		if m.upstream.Upstream == "" {
			branch := m.currentBranch
			if branch == "" {
				branch = m.repo.BranchName()
			}
			m.pushConfirm = true
			var clearCmd tea.Cmd
			m, clearCmd = m.setStatus("press P again to push --set-upstream origin "+branch, true)
			return m, clearCmd
		}
		m.pushConfirm = true
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("press P again to push to "+m.upstream.Upstream, true)
		return m, clearCmd
	}
	m.pushConfirm = false

	if msg.String() == "x" {
		if len(m.files) > 0 && m.cursor < len(m.files) {
			f := m.files[m.cursor]
			if f.change.Staged {
				var clearCmd tea.Cmd
				m, clearCmd = m.setStatus("Cannot discard staged file", true)
				return m, clearCmd
			}
			if m.discardConfirm {
				m.discardConfirm = false
				return m, func() tea.Msg {
					_ = m.repo.DiscardFile(f.change.Path, f.untracked)
					return m.buildRefreshedFiles()
				}
			}
			m.discardConfirm = true
			var clearCmd tea.Cmd
			m, clearCmd = m.setStatus("press x again to discard changes", true)
			return m, clearCmd
		}
	}
	m.discardConfirm = false

	switch msg.String() {
	case "?":
		m.prevMode = m.mode
		m.mode = modeHelp
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.files)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = max(0, len(m.files)-1)
	case "enter", "l", "right":
		m.mode = modeDiff
		return m, nil
	case "e":
		if m.cursor < len(m.files) {
			m.SelectedFile = m.files[m.cursor].change.Path
		}
		return m, tea.Quit
	case "tab":
		return m.toggleStage()
	case "a":
		return m.stageAll()
	case "c":
		return m.enterCommitMode()
	case "b":
		return m.enterBranchMode()
	case "s":
		m.splitDiff = !m.splitDiff
		m.prevCurs = -1
		m.lastDiffContent = ""
		return m, tea.Batch(m.loadDiffCmd(true), m.saveConfigCmd())
	case "w":
		m.wrapDiff = !m.wrapDiff
		m.prevCurs = -1
		m.lastDiffContent = ""
		return m, tea.Batch(m.loadDiffCmd(true), m.saveConfigCmd())
	case "F":
		if m.upstream.Upstream == "" {
			var clearCmd tea.Cmd
			m, clearCmd = m.setStatus("no upstream configured", true)
			return m, clearCmd
		}
		m.statusMsg = "pulling..."
		return m, m.pullCmd()
	}
	if m.cursor != m.prevCurs {
		m.prevCurs = m.cursor
		return m, m.loadDiffCmd(true)
	}
	return m, nil
}

func (m Model) nextFile() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.files)-1 {
		m.cursor++
		m.prevCurs = m.cursor
		return m, m.loadDiffCmd(true)
	}
	return m, nil
}

func (m Model) prevFile() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
		m.prevCurs = m.cursor
		return m, m.loadDiffCmd(true)
	}
	return m, nil
}

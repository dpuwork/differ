package ui

import tea "charm.land/bubbletea/v2"

// Diff mode key handling and viewport delegation.

func (m Model) updateDiffMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	
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
	case "esc", "h", "left":
		m.mode = modeFileList
		return m, nil
	case "n":
		return m.nextFile()
	case "p":
		return m.prevFile()
	case "e":
		if m.cursor < len(m.files) {
			m.SelectedFile = m.files[m.cursor].change.Path
		}
		return m, tea.Quit
	case "b":
		return m.enterBranchMode()
	case "tab":
		return m.toggleStage()
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
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

package ui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// Update stays dispatcher-only; behavior lives in focused modules.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearStatusMsg:
		if msg.id == m.statusMsgID {
			m.statusMsg = ""
		}
		return m, nil
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case tickMsg:
		return m.handleTick()
	case diffLoadedMsg:
		return m.handleDiffLoaded(msg)
	case filesRefreshedMsg:
		return m.handleFilesRefreshed(msg)
	case commitDoneMsg:
		return m.handleCommitDone(msg)
	case branchesLoadedMsg:
		return m.handleBranchesLoaded(msg)
	case branchSwitchedMsg:
		return m.handleBranchSwitched(msg)
	case branchCreatedMsg:
		return m.handleBranchCreated(msg)
	case upstreamStatusMsg:
		return m.handleUpstreamStatus(msg)
	case pushDoneMsg:
		return m.handlePushDone(msg)
	case pullDoneMsg:
		return m.handlePullDone(msg)
	case tickFetchMsg:
		return m.handleTickFetch()
	case fetchRemoteDoneMsg:
		return m.handleFetchRemoteDone(msg)
	case savePrefDoneMsg:
		if msg.err != nil {
			var clearCmd tea.Cmd
			m, clearCmd = m.setStatus("config save failed", true)
			return m, clearCmd
		}
		return m, nil
	case tea.MouseMsg:
		if !m.ready {
			return m, nil
		}
		mouseEvent := msg.Mouse()
		if _, isClick := msg.(tea.MouseClickMsg); isClick && mouseEvent.Button == tea.MouseLeft {
			return m.handleMouseLeftClick(mouseEvent)
		}
		return m, nil
	case tea.KeyPressMsg:
		if !m.ready {
			return m, nil
		}
		switch m.mode {
		case modeFileList:
			return m.updateFileListMode(msg)
		case modeDiff:
			return m.updateDiffMode(msg)
		case modeCommit:
			return m.updateCommitMode(msg)
		case modeBranchPicker:
			return m.updateBranchMode(msg)
		case modeHelp:
			return m.updateHelpMode(msg)
		}
	}
	return m, nil
}

func (m Model) handleMouseLeftClick(msg tea.Mouse) (tea.Model, tea.Cmd) {
	// If it's a help popup or we are typing, ignore
	if m.mode == modeHelp || m.mode == modeCommit || m.mode == modeBranchPicker {
		return m, nil
	}

	isLeftPane := msg.X <= fileListWidth+1
	isRightPane := msg.X > fileListWidth+2

	var cmd tea.Cmd

	if isLeftPane {
		if m.mode != modeFileList {
			m.mode = modeFileList
		}
		
		// Determine which file was clicked
		// The list starts at Y=1 (due to top border)
		// We need to account for the "STAGED" and "CHANGES" headers
		clickY := msg.Y - 1 // 0-indexed relative to content area
		if clickY >= 0 && clickY < m.contentHeight() {
			stagedCount := 0
			for _, f := range m.files {
				if f.change.Staged {
					stagedCount++
				}
			}

			renderedLines := 0
			clickedIndex := -1

			if stagedCount > 0 {
				// STAGED header
				if clickY == renderedLines {
					return m, nil // Clicked header
				}
				renderedLines++

				for i := 0; i < stagedCount; i++ {
					if clickY == renderedLines {
						clickedIndex = i
						break
					}
					renderedLines++
				}
			}

			if clickedIndex == -1 && len(m.files) > stagedCount {
				if stagedCount > 0 {
					if clickY == renderedLines {
						return m, nil // Clicked empty margin
					}
					renderedLines++ // Account for MarginTop(1) on CHANGES header
				}
				// CHANGES header
				if clickY == renderedLines {
					return m, nil // Clicked header
				}
				renderedLines++

				for i := stagedCount; i < len(m.files); i++ {
					if clickY == renderedLines {
						clickedIndex = i
						break
					}
					renderedLines++
				}
			}

			if clickedIndex != -1 {
				m.cursor = clickedIndex
				m.prevCurs = clickedIndex
				cmd = m.loadDiffCmd(true)
			}
		}

	} else if isRightPane {
		if m.mode != modeDiff {
			m.mode = modeDiff
		}
	}

	return m, cmd
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.refreshTheme()
	m.viewport = viewport.New(viewport.WithWidth(m.diffWidth()), viewport.WithHeight(m.contentHeight()))
	m.lastDiffContent = ""
	m.ready = m.ready || len(m.files) == 0
	return m, m.loadDiffCmd(true)
}

func (m Model) handleDiffLoaded(msg diffLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.index != m.cursor {
		return m, nil
	}
	m.ready = true
	if msg.content == m.lastDiffContent {
		return m, nil
	}
	m.lastDiffContent = msg.content
	m.viewport.SetContent(msg.content)
	if msg.resetScroll {
		m.viewport.GotoTop()
	}
	return m, nil
}

func (m Model) handleFilesRefreshed(msg filesRefreshedMsg) (tea.Model, tea.Cmd) {
	if filesEqual(m.files, msg.files) {
		return m, m.loadDiffCmd(false)
	}
	m.files = msg.files
	if m.cursor >= len(m.files) {
		m.cursor = max(0, len(m.files)-1)
	}
	m.prevCurs = -1
	m.lastDiffContent = ""
	if len(m.files) == 0 {
		m.viewport.SetContent("")
		return m, nil
	}
	return m, m.loadDiffCmd(true)
}

func (m Model) handleCommitDone(msg commitDoneMsg) (tea.Model, tea.Cmd) {
	m.mode = modeFileList
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("commit failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	var clearCmd tea.Cmd
	m, clearCmd = m.setStatus("committed!", true)
	m.commitInput.Reset()
	return m, tea.Batch(m.refreshFilesCmd(), clearCmd)
}

func (m Model) handleBranchesLoaded(msg branchesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("branch list failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	if len(msg.branches) == 0 {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("no branches", true)
		return m, clearCmd
	}
	m.mode = modeBranchPicker
	m.branches = msg.branches
	m.currentBranch = msg.current
	m.branchCursor = 0
	m.branchOffset = 0
	for i, b := range m.branches {
		if b == msg.current {
			m.branchCursor = i
			break
		}
	}
	m.filteredBranches = nil
	m.branchFilter.Reset()
	m.branchFilter.Focus()
	return m, textinput.Blink
}

func (m Model) handleBranchSwitched(msg branchSwitchedMsg) (tea.Model, tea.Cmd) {
	m.mode = modeFileList
	m.filteredBranches = nil
	m.branchFilter.Reset()
	m.branchFilter.Blur()
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("switch failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	var clearCmd tea.Cmd
	m, clearCmd = m.setStatus("switched to "+m.repo.BranchName(), true)
	m.prevCurs = -1
	m.cursor = 0
	return m, tea.Batch(m.refreshFilesCmd(), clearCmd)
}

func (m Model) handleBranchCreated(msg branchCreatedMsg) (tea.Model, tea.Cmd) {
	m.branchCreating = false
	m.branchInput.Reset()
	if msg.err != nil {
		var clearCmd tea.Cmd
		m, clearCmd = m.setStatus("create failed: "+msg.err.Error(), true)
		return m, clearCmd
	}
	m.mode = modeFileList
	var clearCmd tea.Cmd
	m, clearCmd = m.setStatus("created & switched to "+msg.name, true)
	m.prevCurs = -1
	m.cursor = 0
	return m, tea.Batch(m.refreshFilesCmd(), clearCmd)
}

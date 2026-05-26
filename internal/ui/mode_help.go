package ui

import tea "charm.land/bubbletea/v2"

func (m Model) updateHelpMode(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc", "?":
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case snapshotMsg:
		m.snapshot = msg.snapshot
		m.buildItems()
		return m, waitForEvent(m.eventCh)

	case switchPaneMsg:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyMsg:
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeSpawn:
			return m.updateSpawn(msg)
		default:
			return m.updateNormal(msg)
		}
	}

	return m, nil
}

// updateNormal handles key presses in normal mode.
func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "j", "down":
		m.moveDown()
		return m, nil

	case "k", "up":
		m.moveUp()
		return m, nil

	case "enter":
		if pa := m.selectedAgent(); pa != nil {
			return m, m.switchToPane(pa.Pane.PaneID)
		}
		return m, nil

	case "/":
		m.mode = modeFilter
		m.filterText = ""
		return m, nil

	case "n":
		m.mode = modeSpawn
		m.spawnCursor = 0
		return m, nil

	case "esc":
		if m.filterText != "" {
			m.filterText = ""
			m.buildItems()
		}
		return m, nil
	}

	return m, nil
}

// updateFilter handles key presses in filter mode.
func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.filterText = ""
		m.buildItems()
		return m, nil

	case "enter":
		m.mode = modeNormal
		// Keep filter applied.
		return m, nil

	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.buildItems()
		}
		return m, nil

	default:
		// Only accept printable single runes.
		if len(msg.Runes) > 0 {
			m.filterText += string(msg.Runes)
			m.buildItems()
		}
		return m, nil
	}
}

// updateSpawn handles key presses in spawn dialog mode.
func (m Model) updateSpawn(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeNormal
		return m, nil

	case "j", "down":
		if m.spawnCursor < len(m.spawnOptions)-1 {
			m.spawnCursor++
		}
		return m, nil

	case "k", "up":
		if m.spawnCursor > 0 {
			m.spawnCursor--
		}
		return m, nil

	case "enter":
		if m.spawnCursor >= 0 && m.spawnCursor < len(m.spawnOptions) {
			opt := m.spawnOptions[m.spawnCursor]
			m.mode = modeNormal
			return m, spawnAgent(m.tmuxClient, opt.command)
		}
		return m, nil
	}

	return m, nil
}

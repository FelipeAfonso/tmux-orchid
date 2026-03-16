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

	case spawnDoneMsg:
		// Agent was spawned (or failed). Stay in normal mode;
		// the state manager will detect it on next poll.
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeSpawnAgent:
			return m.updateSpawnAgent(msg)
		case modeSpawnDir:
			return m.updateSpawnDir(msg)
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
		m.mode = modeSpawnAgent
		m.spawnCursor = 0
		m.spawnPicked = nil
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
		return m, nil

	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.buildItems()
		}
		return m, nil

	default:
		if len(msg.Runes) > 0 {
			m.filterText += string(msg.Runes)
			m.buildItems()
		}
		return m, nil
	}
}

// updateSpawnAgent handles step 1: picking which agent to spawn.
func (m Model) updateSpawnAgent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeNormal
		return m, nil

	case "j", "down":
		if m.spawnCursor < len(m.spawnAgents)-1 {
			m.spawnCursor++
		}
		return m, nil

	case "k", "up":
		if m.spawnCursor > 0 {
			m.spawnCursor--
		}
		return m, nil

	case "enter":
		if m.spawnCursor >= 0 && m.spawnCursor < len(m.spawnAgents) {
			picked := m.spawnAgents[m.spawnCursor]
			m.spawnPicked = &picked
			m.spawnDirs = m.collectSpawnDirs()
			m.spawnDirIdx = 0

			// If only one directory, skip the directory step.
			if len(m.spawnDirs) == 1 {
				m.mode = modeNormal
				return m, m.doSpawn(picked, m.spawnDirs[0])
			}

			m.mode = modeSpawnDir
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// updateSpawnDir handles step 2: picking which project directory.
func (m Model) updateSpawnDir(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Go back to agent selection.
		m.mode = modeSpawnAgent
		m.spawnPicked = nil
		return m, nil

	case "q":
		m.mode = modeNormal
		return m, nil

	case "j", "down":
		if m.spawnDirIdx < len(m.spawnDirs)-1 {
			m.spawnDirIdx++
		}
		return m, nil

	case "k", "up":
		if m.spawnDirIdx > 0 {
			m.spawnDirIdx--
		}
		return m, nil

	case "enter":
		if m.spawnPicked != nil && m.spawnDirIdx >= 0 && m.spawnDirIdx < len(m.spawnDirs) {
			agent := *m.spawnPicked
			dir := m.spawnDirs[m.spawnDirIdx]
			m.mode = modeNormal
			m.spawnPicked = nil
			return m, m.doSpawn(agent, dir)
		}
		return m, nil
	}

	return m, nil
}

package tui

import (
	"log/slog"

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

	case switchDoneMsg:
		// The tmux client was switched to the agent's pane.
		// The dashboard stays running; the user returns via keybind.
		if msg.err != nil {
			slog.Warn("switch to pane failed", "pane", msg.paneID, "error", msg.err)
		}
		return m, nil

	case spawnDoneMsg:
		// Agent was spawned (or failed). Stay in normal mode;
		// the state manager will detect it on next poll.
		if msg.err != nil {
			slog.Warn("spawn failed", "error", msg.err)
		}
		return m, nil

	case paneCaptureTickMsg:
		// Schedule the next tick unconditionally.
		nextTick := paneCaptureTickCmd()
		pa := m.selectedAgent()
		if pa == nil {
			return m, nextTick
		}
		paneID := pa.Pane.PaneID
		if paneID == "" {
			return m, nextTick
		}
		// Track which pane we are capturing; clear stale content
		// if the user moved to a different agent.
		if paneID != m.panePaneID {
			m.paneContent = ""
			m.panePaneID = paneID
		}
		return m, tea.Batch(nextTick, m.capturePaneCmd(paneID))

	case paneCaptureMsg:
		// Only apply if the result still matches the selected pane.
		if msg.paneID != m.panePaneID {
			return m, nil
		}
		if msg.err != nil {
			slog.Debug("pane capture failed", "pane", msg.paneID, "error", msg.err)
			return m, nil
		}
		m.paneContent = msg.content
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeFilter:
			return m.updateFilter(msg)
		case modeSpawnAgent:
			return m.updateSpawnAgent(msg)
		case modeSpawnSession:
			return m.updateSpawnSession(msg)
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
		prev := m.panePaneID
		m.moveDown()
		m.clearPaneIfChanged(prev)
		return m, nil

	case "k", "up":
		prev := m.panePaneID
		m.moveUp()
		m.clearPaneIfChanged(prev)
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
			m.spawnSessions = m.collectSpawnSessions()
			m.spawnSessionIdx = 0

			if len(m.spawnSessions) == 0 {
				// No sessions available; cancel spawn.
				slog.Warn("no tmux sessions available for spawning")
				m.mode = modeNormal
				m.spawnPicked = nil
				return m, nil
			}

			// If only one session, skip the session selection step.
			if len(m.spawnSessions) == 1 {
				m.mode = modeNormal
				return m, m.doSpawn(picked, m.spawnSessions[0])
			}

			m.mode = modeSpawnSession
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// updateSpawnSession handles step 2: picking which tmux session to spawn into.
func (m Model) updateSpawnSession(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.spawnSessionIdx < len(m.spawnSessions)-1 {
			m.spawnSessionIdx++
		}
		return m, nil

	case "k", "up":
		if m.spawnSessionIdx > 0 {
			m.spawnSessionIdx--
		}
		return m, nil

	case "enter":
		if m.spawnPicked != nil && m.spawnSessionIdx >= 0 && m.spawnSessionIdx < len(m.spawnSessions) {
			agent := *m.spawnPicked
			session := m.spawnSessions[m.spawnSessionIdx]
			m.mode = modeNormal
			m.spawnPicked = nil
			return m, m.doSpawn(agent, session)
		}
		return m, nil
	}

	return m, nil
}

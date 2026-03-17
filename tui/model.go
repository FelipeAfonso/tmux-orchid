// Package tui implements the Bubble Tea TUI for tmux-orchid.
package tui

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomalyco/tmux-orchid/spawner"
	"github.com/anomalyco/tmux-orchid/state"
	"github.com/anomalyco/tmux-orchid/tmux"
)

// mode tracks which UI mode we are in.
type mode int

const (
	modeNormal mode = iota
	modeFilter
	modeSpawnAgent // step 1: pick agent
	modeSpawnDir   // step 2: pick project directory
)

// sidebarItem is a flattened row in the sidebar, either a project header
// or an agent entry.
type sidebarItem struct {
	isProject bool
	project   *state.Project
	agent     *state.PaneAgent
}

// Model is the top-level Bubble Tea model.
type Model struct {
	// Dependencies.
	manager    *state.Manager
	tmuxClient *tmux.Client
	spawner    *spawner.Spawner
	eventCh    chan state.Event

	// State.
	snapshot *state.Snapshot
	items    []sidebarItem // flattened sidebar items
	cursor   int           // index into items (only agent rows are selectable)
	mode     mode
	width    int
	height   int

	// Filter.
	filterText string

	// Spawn dialog.
	spawnAgents []spawner.AgentDef
	spawnCursor int
	spawnDirs   []string // project directories to choose from
	spawnDirIdx int
	spawnPicked *spawner.AgentDef // chosen agent (between steps)

	// Quit flag.
	quitting bool
}

// New creates a new TUI model wired to the given state manager and tmux client.
func New(mgr *state.Manager, tc *tmux.Client) Model {
	ch := make(chan state.Event, 128)
	mgr.Subscribe(ch)

	agents := spawner.Available(spawner.DefaultAgents())
	if len(agents) == 0 {
		// Fall back to full list if none found on PATH.
		agents = spawner.DefaultAgents()
	}

	return Model{
		manager:     mgr,
		tmuxClient:  tc,
		spawner:     spawner.New(tc),
		eventCh:     ch,
		spawnAgents: agents,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEvent(m.eventCh),
	)
}

// snapshotMsg delivers a new snapshot from the state manager.
type snapshotMsg struct {
	snapshot *state.Snapshot
}

// switchDoneMsg is sent after the tmux client has been switched to an
// agent's pane. The dashboard remains running.
type switchDoneMsg struct {
	paneID string
	err    error
}

// spawnDoneMsg is sent after a spawn completes (success or failure).
type spawnDoneMsg struct {
	err error
}

// waitForEvent returns a Cmd that listens on the event channel and
// converts state events into Bubble Tea messages.
func waitForEvent(ch <-chan state.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return nil
		}
		if evt.Kind == state.EventSnapshotUpdated && evt.Snapshot != nil {
			return snapshotMsg{snapshot: evt.Snapshot}
		}
		return snapshotMsg{snapshot: evt.Snapshot}
	}
}

// buildItems flattens projects and agents into sidebar rows, applying the
// current filter.
func (m *Model) buildItems() {
	m.items = nil
	if m.snapshot == nil {
		return
	}

	for i := range m.snapshot.Projects {
		proj := &m.snapshot.Projects[i]

		var matchingAgents []*state.PaneAgent
		for j := range proj.Agents {
			if m.matchesFilter(&proj.Agents[j]) {
				matchingAgents = append(matchingAgents, &proj.Agents[j])
			}
		}

		if len(matchingAgents) == 0 {
			continue
		}

		m.items = append(m.items, sidebarItem{
			isProject: true,
			project:   proj,
		})

		for _, pa := range matchingAgents {
			m.items = append(m.items, sidebarItem{
				isProject: false,
				agent:     pa,
			})
		}
	}

	m.clampCursor()
}

// matchesFilter returns true if the agent matches the current filter text.
func (m *Model) matchesFilter(pa *state.PaneAgent) bool {
	if m.filterText == "" {
		return true
	}
	text := string(pa.Agent.Type) + " " + pa.Pane.SessionName + " " + pa.Pane.WindowName + " " + pa.Agent.CWD
	return containsFold(text, m.filterText)
}

// clampCursor ensures cursor points to a valid agent row.
func (m *Model) clampCursor() {
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.items[m.cursor].isProject {
		m.moveDown()
	}
}

// moveDown moves cursor to the next agent row.
func (m *Model) moveDown() {
	for i := m.cursor + 1; i < len(m.items); i++ {
		if !m.items[i].isProject {
			m.cursor = i
			return
		}
	}
	for i := 0; i < len(m.items); i++ {
		if !m.items[i].isProject {
			m.cursor = i
			return
		}
	}
}

// moveUp moves cursor to the previous agent row.
func (m *Model) moveUp() {
	for i := m.cursor - 1; i >= 0; i-- {
		if !m.items[i].isProject {
			m.cursor = i
			return
		}
	}
	for i := len(m.items) - 1; i >= 0; i-- {
		if !m.items[i].isProject {
			m.cursor = i
			return
		}
	}
}

// selectedAgent returns the currently selected agent, or nil.
func (m *Model) selectedAgent() *state.PaneAgent {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	if item.isProject {
		return nil
	}
	return item.agent
}

// switchToPane switches the tmux client to the given pane without
// quitting the dashboard. The user is moved to the agent's session/pane
// and can return to the dashboard via the keybind.
func (m *Model) switchToPane(paneID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.tmuxClient.SwitchToPane(ctx, paneID)
		return switchDoneMsg{paneID: paneID, err: err}
	}
}

// spawnDirs collects unique project directories from the current snapshot,
// plus the current working directory.
func (m *Model) collectSpawnDirs() []string {
	seen := make(map[string]bool)
	var dirs []string

	// Current working directory first.
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		dirs = append(dirs, cwd)
		seen[cwd] = true
	}

	// Project roots from the snapshot.
	if m.snapshot != nil {
		for _, p := range m.snapshot.Projects {
			if p.GitRoot != "" && !seen[p.GitRoot] {
				dirs = append(dirs, p.GitRoot)
				seen[p.GitRoot] = true
			}
		}
	}

	return dirs
}

// doSpawn creates the tmux session via the spawner package.
func (m *Model) doSpawn(agent spawner.AgentDef, dir string) tea.Cmd {
	sp := m.spawner
	return func() tea.Msg {
		ctx := context.Background()
		_, err := sp.Spawn(ctx, spawner.Request{
			Agent: agent,
			Dir:   dir,
		})
		return spawnDoneMsg{err: err}
	}
}

// Package tui implements the Bubble Tea TUI for tmux-orchid.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomalyco/tmux-orchid/state"
	"github.com/anomalyco/tmux-orchid/tmux"
)

// mode tracks which UI mode we are in.
type mode int

const (
	modeNormal mode = iota
	modeFilter
	modeSpawn
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
	spawnOptions []spawnOption
	spawnCursor  int

	// Switch-to-pane action: set when user presses Enter.
	switchPaneID string

	// Quit flag.
	quitting bool
}

// spawnOption is a chooseable agent type in the spawn dialog.
type spawnOption struct {
	label   string
	command string
}

var defaultSpawnOptions = []spawnOption{
	{label: "Claude Code", command: "claude"},
	{label: "OpenCode", command: "opencode"},
	{label: "Aider", command: "aider"},
	{label: "Codex", command: "codex"},
	{label: "Gemini CLI", command: "gemini"},
	{label: "Goose", command: "goose"},
	{label: "Amp", command: "amp"},
}

// New creates a new TUI model wired to the given state manager and tmux client.
func New(mgr *state.Manager, tc *tmux.Client) Model {
	ch := make(chan state.Event, 128)
	mgr.Subscribe(ch)

	return Model{
		manager:      mgr,
		tmuxClient:   tc,
		eventCh:      ch,
		spawnOptions: defaultSpawnOptions,
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

// switchPaneMsg signals the program should quit and switch tmux pane.
type switchPaneMsg struct {
	paneID string
}

// spawnAgentMsg signals a new agent should be spawned.
type spawnAgentMsg struct {
	command string
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
		// For granular events, we still want to keep listening.
		// Re-read the channel by returning another wait.
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

		// Collect agents matching the filter.
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

	// Clamp cursor.
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
	// If cursor lands on a project header, move to next agent.
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
	// No agent below, stay put or find first agent.
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
	// No agent above, wrap to last agent.
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

// switchToPane issues a tmux select-window + select-pane so the user
// lands on the agent's pane after the TUI quits.
func (m *Model) switchToPane(paneID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// select-window first, then select-pane. Both target the pane ID.
		_ = m.tmuxClient.SwitchToPane(ctx, paneID)
		return switchPaneMsg{paneID: paneID}
	}
}

// spawnAgent opens a new tmux window with the given command.
func spawnAgent(tc *tmux.Client, command string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, _ = tc.NewWindow(ctx, command)
		return nil
	}
}

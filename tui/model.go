// Package tui implements the Bubble Tea TUI for tmux-orchid.
package tui

import (
	"context"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomalyco/tmux-orchid/spawner"
	"github.com/anomalyco/tmux-orchid/state"
	"github.com/anomalyco/tmux-orchid/tmux"
)

// paneCaptureInterval is how often we re-capture the selected agent's pane.
const paneCaptureInterval = 300 * time.Millisecond

// mode tracks which UI mode we are in.
type mode int

const (
	modeNormal mode = iota
	modeFilter
	modeSpawnAgent   // step 1: pick agent
	modeSpawnSession // step 2: pick target session
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

	// Pane capture (live terminal preview).
	paneContent string // ANSI-escaped content of the selected pane
	panePaneID  string // pane ID currently being captured

	// Spawn dialog.
	spawnAgents     []spawner.AgentDef
	spawnCursor     int
	spawnSessions   []string // tmux sessions to choose from
	spawnSessionIdx int
	spawnPicked     *spawner.AgentDef // chosen agent (between steps)

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
		paneCaptureTickCmd(),
	)
}

// paneCaptureTickCmd returns a tick command that fires paneCaptureTickMsg
// at the configured interval.
func paneCaptureTickCmd() tea.Cmd {
	return tea.Tick(paneCaptureInterval, func(_ time.Time) tea.Msg {
		return paneCaptureTickMsg{}
	})
}

// capturePaneCmd runs tmux capture-pane with ANSI escapes for the given pane.
func (m *Model) capturePaneCmd(paneID string) tea.Cmd {
	tc := m.tmuxClient
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		content, err := tc.CapturePaneANSI(ctx, paneID)
		return paneCaptureMsg{paneID: paneID, content: content, err: err}
	}
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

// paneCaptureTickMsg triggers a pane content capture for the selected agent.
type paneCaptureTickMsg struct{}

// paneCaptureMsg delivers captured pane content (with ANSI escapes).
type paneCaptureMsg struct {
	paneID  string
	content string
	err     error
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

// clearPaneIfChanged resets cached pane content when the selected agent
// has changed. prevPaneID is the pane ID before the cursor moved.
func (m *Model) clearPaneIfChanged(prevPaneID string) {
	pa := m.selectedAgent()
	newID := ""
	if pa != nil {
		newID = pa.Pane.PaneID
	}
	if newID != prevPaneID {
		m.paneContent = ""
		m.panePaneID = newID
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

// collectSpawnSessions returns the names of all tmux sessions, excluding
// the session that the dashboard itself is running in.
func (m *Model) collectSpawnSessions() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessions, err := m.tmuxClient.ListSessions(ctx)
	if err != nil {
		slog.Warn("failed to list tmux sessions", "error", err)
		return nil
	}

	// Exclude the dashboard's own session.
	own, err := m.tmuxClient.CurrentSession(ctx)
	if err != nil {
		slog.Warn("failed to get current session", "error", err)
	}

	var out []string
	for _, s := range sessions {
		if s != own {
			out = append(out, s)
		}
	}
	return out
}

// doSpawn creates a new tmux window in the target session via the spawner.
func (m *Model) doSpawn(agent spawner.AgentDef, session string) tea.Cmd {
	sp := m.spawner
	return func() tea.Msg {
		ctx := context.Background()
		_, err := sp.Spawn(ctx, spawner.Request{
			Agent:         agent,
			TargetSession: session,
		})
		return spawnDoneMsg{err: err}
	}
}

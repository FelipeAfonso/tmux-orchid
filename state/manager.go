package state

import (
	"context"
	"log/slog"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/FelipeAfonso/tmux-orchid/detector"
	"github.com/FelipeAfonso/tmux-orchid/tmux"
)

// PaneLister lists all tmux panes. Satisfied by tmux.Client.
type PaneLister interface {
	ListPanes(ctx context.Context) ([]tmux.Pane, error)
}

// PaneCapturer captures the content of a tmux pane. Satisfied by tmux.Client.
type PaneCapturer interface {
	CapturePane(ctx context.Context, paneID string) (string, error)
}

// Source combines the abilities to list and capture panes.
type Source interface {
	PaneLister
	PaneCapturer
}

// Manager polls tmux at a configurable interval, detects agents, groups
// them into projects by git root, diffs against the previous state, and
// notifies registered listeners.
type Manager struct {
	source       Source
	pollInterval time.Duration
	sessions     []string // optional session name filter

	mu       sync.RWMutex
	current  *Snapshot
	gitCache *gitRootCache

	listenersMu sync.RWMutex
	listeners   []chan<- Event

	// gitRootFunc can be overridden for testing. Defaults to gitCache.Resolve.
	gitRootFunc func(string) string
}

// NewManager creates a new state manager that polls the given source at the
// specified interval. If sessions is non-empty, only panes from those tmux
// session names are considered.
func NewManager(src Source, pollInterval time.Duration, sessions []string) *Manager {
	gc := newGitRootCache()
	return &Manager{
		source:       src,
		pollInterval: pollInterval,
		sessions:     sessions,
		gitCache:     gc,
		gitRootFunc:  gc.Resolve,
	}
}

// Subscribe registers a channel to receive state change events. The caller
// must drain the channel to avoid blocking the manager. Returns an
// unsubscribe function.
func (m *Manager) Subscribe(ch chan<- Event) func() {
	m.listenersMu.Lock()
	m.listeners = append(m.listeners, ch)
	m.listenersMu.Unlock()

	return func() {
		m.listenersMu.Lock()
		defer m.listenersMu.Unlock()
		for i, l := range m.listeners {
			if l == ch {
				m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
				return
			}
		}
	}
}

// Current returns the most recent snapshot. May be nil before the first poll.
func (m *Manager) Current() *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) {
	// Do an immediate poll on start.
	m.poll(ctx)

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

// PollOnce performs a single poll cycle. Useful for testing.
func (m *Manager) PollOnce(ctx context.Context) {
	m.poll(ctx)
}

// poll performs one scan cycle: list panes, detect agents, build snapshot,
// diff, and notify.
func (m *Manager) poll(ctx context.Context) {
	panes, err := m.source.ListPanes(ctx)
	if err != nil {
		slog.Warn("failed to list panes", "error", err)
		return
	}

	panes = m.filterSessions(panes)

	agents := detector.DetectAll(ctx, panes, m.source)

	snap := m.buildSnapshot(panes, agents)

	m.mu.RLock()
	old := m.current
	m.mu.RUnlock()

	events := diffSnapshots(old, snap)

	m.mu.Lock()
	m.current = snap
	m.mu.Unlock()

	// Always emit a snapshot-updated event.
	events = append(events, Event{
		Kind:     EventSnapshotUpdated,
		Snapshot: snap,
	})

	m.emit(events)
}

// filterSessions limits panes to those belonging to the configured session
// names. If no filter is set, all panes pass through.
func (m *Manager) filterSessions(panes []tmux.Pane) []tmux.Pane {
	if len(m.sessions) == 0 {
		return panes
	}

	allowed := make(map[string]bool, len(m.sessions))
	for _, s := range m.sessions {
		allowed[s] = true
	}

	filtered := make([]tmux.Pane, 0, len(panes))
	for _, p := range panes {
		if allowed[p.SessionName] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// buildSnapshot groups detected agents into projects by git root.
func (m *Manager) buildSnapshot(panes []tmux.Pane, agents map[string]detector.AgentInfo) *Snapshot {
	paneMap := make(map[string]tmux.Pane, len(panes))
	for _, p := range panes {
		paneMap[p.PaneID] = p
	}

	// Group agents by git root.
	byRoot := make(map[string][]PaneAgent)
	allAgents := make(map[string]PaneAgent, len(agents))

	for paneID, info := range agents {
		pane := paneMap[paneID]
		pa := PaneAgent{Pane: pane, Agent: info}
		allAgents[paneID] = pa

		root := m.resolveGitRoot(info.CWD)
		byRoot[root] = append(byRoot[root], pa)
	}

	// Build sorted project list.
	projects := make([]Project, 0, len(byRoot))
	for root, pas := range byRoot {
		// Sort agents within project by pane ID for determinism.
		sort.Slice(pas, func(i, j int) bool {
			return pas[i].Pane.PaneID < pas[j].Pane.PaneID
		})
		projects = append(projects, Project{
			GitRoot: root,
			Name:    projectName(root),
			Agents:  pas,
		})
	}
	// Sort projects by name for deterministic ordering.
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return &Snapshot{
		Projects:  projects,
		Agents:    allAgents,
		Timestamp: time.Now(),
	}
}

// resolveGitRoot uses the git root function (overridable for testing).
func (m *Manager) resolveGitRoot(dir string) string {
	if m.gitRootFunc != nil {
		return m.gitRootFunc(dir)
	}
	return dir
}

// emit sends events to all registered listeners. Non-blocking: if a
// listener's channel is full, the event is dropped with a warning.
func (m *Manager) emit(events []Event) {
	m.listenersMu.RLock()
	defer m.listenersMu.RUnlock()

	for _, evt := range events {
		for _, ch := range m.listeners {
			select {
			case ch <- evt:
			default:
				slog.Debug("dropping event for slow listener", "kind", evt.Kind)
			}
		}
	}
}

// ProjectForPath returns the project containing the given absolute path,
// or nil if no project matches.
func (s *Snapshot) ProjectForPath(absPath string) *Project {
	if s == nil {
		return nil
	}
	clean := filepath.Clean(absPath)
	for i := range s.Projects {
		if clean == s.Projects[i].GitRoot || isSubpath(s.Projects[i].GitRoot, clean) {
			return &s.Projects[i]
		}
	}
	return nil
}

// isSubpath returns true if child is under parent.
func isSubpath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// Rel returns ".." prefixes when child is not under parent.
	return len(rel) > 0 && rel[0] != '.'
}

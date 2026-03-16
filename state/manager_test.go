package state

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/anomalyco/tmux-orchid/detector"
	"github.com/anomalyco/tmux-orchid/tmux"
)

// mockSource implements Source for testing.
type mockSource struct {
	mu      sync.Mutex
	panes   []tmux.Pane
	content map[string]string // paneID -> captured content
	err     error
}

func (m *mockSource) ListPanes(_ context.Context) ([]tmux.Pane, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.panes, nil
}

func (m *mockSource) CapturePane(_ context.Context, paneID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	return m.content[paneID], nil
}

func (m *mockSource) setPanes(panes []tmux.Pane, content map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panes = panes
	m.content = content
}

func TestManagerPollOnce(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project/a"},
			{PaneID: "%1", PanePID: 200, CurrentCommand: "bash", CurrentPath: "/home/user"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
			"%1": "$ ls\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	// Override git root to return cwd directly (avoid filesystem dependency).
	mgr.gitRootFunc = func(dir string) string { return dir }

	mgr.PollOnce(context.Background())

	snap := mgr.Current()
	if snap == nil {
		t.Fatal("Current() returned nil after PollOnce")
	}

	// Only claude should be detected (bash is not an agent).
	if len(snap.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(snap.Agents))
	}

	pa, ok := snap.Agents["%0"]
	if !ok {
		t.Fatal("expected agent in pane %0")
	}
	if pa.Agent.Type != detector.AgentClaudeCode {
		t.Errorf("agent type = %q, want claude-code", pa.Agent.Type)
	}
	if pa.Agent.Status != detector.StatusThinking {
		t.Errorf("agent status = %q, want thinking", pa.Agent.Status)
	}
}

func TestManagerProjectGrouping(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/projects/alpha/src"},
			{PaneID: "%1", PanePID: 200, CurrentCommand: "aider", CurrentPath: "/projects/alpha/tests"},
			{PaneID: "%2", PanePID: 300, CurrentCommand: "codex", CurrentPath: "/projects/beta"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
			"%1": "aider> \n",
			"%2": "codex> \n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	// Map both alpha paths to same root, beta to different root.
	mgr.gitRootFunc = func(dir string) string {
		switch dir {
		case "/projects/alpha/src", "/projects/alpha/tests":
			return "/projects/alpha"
		case "/projects/beta":
			return "/projects/beta"
		default:
			return dir
		}
	}

	mgr.PollOnce(context.Background())

	snap := mgr.Current()
	if snap == nil {
		t.Fatal("nil snapshot")
	}

	if len(snap.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(snap.Projects))
	}

	// Projects should be sorted by name.
	if snap.Projects[0].Name != "alpha" {
		t.Errorf("first project = %q, want alpha", snap.Projects[0].Name)
	}
	if snap.Projects[1].Name != "beta" {
		t.Errorf("second project = %q, want beta", snap.Projects[1].Name)
	}

	// Alpha should have 2 agents.
	if len(snap.Projects[0].Agents) != 2 {
		t.Errorf("alpha agents = %d, want 2", len(snap.Projects[0].Agents))
	}
	// Beta should have 1 agent.
	if len(snap.Projects[1].Agents) != 1 {
		t.Errorf("beta agents = %d, want 1", len(snap.Projects[1].Agents))
	}
}

func TestManagerSessionFilter(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/a", SessionName: "dev"},
			{PaneID: "%1", PanePID: 200, CurrentCommand: "aider", CurrentPath: "/b", SessionName: "work"},
			{PaneID: "%2", PanePID: 300, CurrentCommand: "codex", CurrentPath: "/c", SessionName: "personal"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
			"%1": "aider> \n",
			"%2": "codex> \n",
		},
	}

	// Only allow "dev" and "work" sessions.
	mgr := NewManager(src, time.Second, []string{"dev", "work"})
	mgr.gitRootFunc = func(dir string) string { return dir }

	mgr.PollOnce(context.Background())

	snap := mgr.Current()
	if snap == nil {
		t.Fatal("nil snapshot")
	}

	// Only panes from "dev" and "work" should be present.
	if len(snap.Agents) != 2 {
		t.Fatalf("expected 2 agents with filter, got %d", len(snap.Agents))
	}
	if _, ok := snap.Agents["%2"]; ok {
		t.Error("pane %2 from 'personal' session should be filtered out")
	}
}

func TestManagerSubscribeAndNotify(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	ch := make(chan Event, 64)
	unsub := mgr.Subscribe(ch)
	defer unsub()

	mgr.PollOnce(context.Background())

	// Collect events with timeout.
	events := collectEvents(t, ch, 200*time.Millisecond)

	// Should have at least: 1 agent added + 1 snapshot updated.
	hasAdded := false
	hasSnapshot := false
	for _, e := range events {
		switch e.Kind {
		case EventAgentAdded:
			hasAdded = true
		case EventSnapshotUpdated:
			hasSnapshot = true
		}
	}

	if !hasAdded {
		t.Error("expected EventAgentAdded event")
	}
	if !hasSnapshot {
		t.Error("expected EventSnapshotUpdated event")
	}
}

func TestManagerSubscribeStatusChange(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	// First poll to establish baseline.
	mgr.PollOnce(context.Background())

	ch := make(chan Event, 64)
	unsub := mgr.Subscribe(ch)
	defer unsub()

	// Change status.
	src.setPanes(
		[]tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		map[string]string{
			"%0": "Reading file src/main.go\n",
		},
	)

	mgr.PollOnce(context.Background())

	events := collectEvents(t, ch, 200*time.Millisecond)

	hasStatusChange := false
	for _, e := range events {
		if e.Kind == EventAgentStatusChanged && e.PaneID == "%0" {
			hasStatusChange = true
			if e.Agent.Status != detector.StatusToolUse {
				t.Errorf("new status = %q, want tool_use", e.Agent.Status)
			}
		}
	}
	if !hasStatusChange {
		t.Error("expected EventAgentStatusChanged event")
	}
}

func TestManagerSubscribeAgentRemoved(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	// First poll.
	mgr.PollOnce(context.Background())

	ch := make(chan Event, 64)
	unsub := mgr.Subscribe(ch)
	defer unsub()

	// Remove the agent (change command to bash).
	src.setPanes(
		[]tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "bash", CurrentPath: "/project"},
		},
		map[string]string{
			"%0": "$ ls\n",
		},
	)

	mgr.PollOnce(context.Background())

	events := collectEvents(t, ch, 200*time.Millisecond)

	hasRemoved := false
	for _, e := range events {
		if e.Kind == EventAgentRemoved && e.PaneID == "%0" {
			hasRemoved = true
		}
	}
	if !hasRemoved {
		t.Error("expected EventAgentRemoved event")
	}
}

func TestManagerUnsubscribe(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	ch := make(chan Event, 64)
	unsub := mgr.Subscribe(ch)

	// Unsubscribe before polling.
	unsub()

	mgr.PollOnce(context.Background())

	events := collectEvents(t, ch, 100*time.Millisecond)
	if len(events) != 0 {
		t.Errorf("expected 0 events after unsubscribe, got %d", len(events))
	}
}

func TestManagerRunContext(t *testing.T) {
	src := &mockSource{
		panes:   []tmux.Pane{},
		content: map[string]string{},
	}

	mgr := NewManager(src, 50*time.Millisecond, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	ch := make(chan Event, 256)
	unsub := mgr.Subscribe(ch)
	defer unsub()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		mgr.Run(ctx)
		close(done)
	}()

	// Wait for Run to finish.
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}

	// Should have received multiple snapshot events.
	events := collectEvents(t, ch, 100*time.Millisecond)
	snapshots := filterTestEvents(events, EventSnapshotUpdated)
	if len(snapshots) < 2 {
		t.Errorf("expected at least 2 snapshot events from polling, got %d", len(snapshots))
	}
}

func TestManagerCurrentBeforePoll(t *testing.T) {
	src := &mockSource{
		panes:   []tmux.Pane{},
		content: map[string]string{},
	}

	mgr := NewManager(src, time.Second, nil)

	if mgr.Current() != nil {
		t.Error("Current() should be nil before first poll")
	}
}

func TestManagerListPanesError(t *testing.T) {
	src := &mockSource{
		err: context.DeadlineExceeded,
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	ch := make(chan Event, 64)
	unsub := mgr.Subscribe(ch)
	defer unsub()

	mgr.PollOnce(context.Background())

	// Should not crash and snapshot should remain nil.
	if mgr.Current() != nil {
		t.Error("Current() should be nil after failed poll")
	}

	events := collectEvents(t, ch, 100*time.Millisecond)
	if len(events) != 0 {
		t.Errorf("expected 0 events on poll error, got %d", len(events))
	}
}

func TestSnapshotProjectForPath(t *testing.T) {
	snap := &Snapshot{
		Projects: []Project{
			{GitRoot: "/projects/alpha", Name: "alpha"},
			{GitRoot: "/projects/beta", Name: "beta"},
		},
	}

	tests := []struct {
		path     string
		wantName string
		wantNil  bool
	}{
		{"/projects/alpha", "alpha", false},
		{"/projects/alpha/src/main.go", "alpha", false},
		{"/projects/beta", "beta", false},
		{"/projects/gamma", "", true},
		{"/other/path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			p := snap.ProjectForPath(tt.path)
			if tt.wantNil {
				if p != nil {
					t.Errorf("expected nil for %q, got project %q", tt.path, p.Name)
				}
				return
			}
			if p == nil {
				t.Fatalf("expected project for %q, got nil", tt.path)
			}
			if p.Name != tt.wantName {
				t.Errorf("project for %q = %q, want %q", tt.path, p.Name, tt.wantName)
			}
		})
	}
}

func TestSnapshotProjectForPathNil(t *testing.T) {
	var snap *Snapshot
	if snap.ProjectForPath("/any") != nil {
		t.Error("expected nil from nil snapshot")
	}
}

func TestIsSubpath(t *testing.T) {
	tests := []struct {
		parent, child string
		want          bool
	}{
		{"/a", "/a/b", true},
		{"/a", "/a/b/c", true},
		{"/a", "/a", false},
		{"/a", "/b", false},
		{"/a/b", "/a", false},
	}

	for _, tt := range tests {
		got := isSubpath(tt.parent, tt.child)
		if got != tt.want {
			t.Errorf("isSubpath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
		}
	}
}

func TestManagerMultipleListeners(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return dir }

	ch1 := make(chan Event, 64)
	ch2 := make(chan Event, 64)
	unsub1 := mgr.Subscribe(ch1)
	unsub2 := mgr.Subscribe(ch2)
	defer unsub1()
	defer unsub2()

	mgr.PollOnce(context.Background())

	events1 := collectEvents(t, ch1, 200*time.Millisecond)
	events2 := collectEvents(t, ch2, 200*time.Millisecond)

	if len(events1) == 0 {
		t.Error("listener 1 received no events")
	}
	if len(events2) == 0 {
		t.Error("listener 2 received no events")
	}
	if len(events1) != len(events2) {
		t.Errorf("listeners received different event counts: %d vs %d", len(events1), len(events2))
	}
}

func TestManagerAgentsWithinProjectSortedByPaneID(t *testing.T) {
	src := &mockSource{
		panes: []tmux.Pane{
			{PaneID: "%2", PanePID: 300, CurrentCommand: "codex", CurrentPath: "/project"},
			{PaneID: "%0", PanePID: 100, CurrentCommand: "claude", CurrentPath: "/project"},
			{PaneID: "%1", PanePID: 200, CurrentCommand: "aider", CurrentPath: "/project"},
		},
		content: map[string]string{
			"%0": "Thinking...\n",
			"%1": "aider> \n",
			"%2": "codex> \n",
		},
	}

	mgr := NewManager(src, time.Second, nil)
	mgr.gitRootFunc = func(dir string) string { return "/project" }

	mgr.PollOnce(context.Background())

	snap := mgr.Current()
	if snap == nil {
		t.Fatal("nil snapshot")
	}

	if len(snap.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(snap.Projects))
	}

	agents := snap.Projects[0].Agents
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}

	// Should be sorted by pane ID.
	for i := 1; i < len(agents); i++ {
		if agents[i-1].Pane.PaneID >= agents[i].Pane.PaneID {
			t.Errorf("agents not sorted: %q >= %q", agents[i-1].Pane.PaneID, agents[i].Pane.PaneID)
		}
	}
}

// collectEvents drains a channel for the given duration and returns events.
func collectEvents(t *testing.T, ch <-chan Event, timeout time.Duration) []Event {
	t.Helper()
	var events []Event
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case e := <-ch:
			events = append(events, e)
		case <-timer.C:
			return events
		}
	}
}

func filterTestEvents(events []Event, kind EventKind) []Event {
	var out []Event
	for _, e := range events {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

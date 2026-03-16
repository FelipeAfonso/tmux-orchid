package state

import (
	"testing"
	"time"

	"github.com/anomalyco/tmux-orchid/detector"
	"github.com/anomalyco/tmux-orchid/tmux"
)

func TestDiffSnapshotsFromNil(t *testing.T) {
	snap := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
			"%1": {
				Pane:  tmux.Pane{PaneID: "%1"},
				Agent: detector.AgentInfo{Type: detector.AgentAider, Status: detector.StatusThinking},
			},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(nil, snap)

	added := filterEvents(events, EventAgentAdded)
	if len(added) != 2 {
		t.Errorf("expected 2 added events, got %d", len(added))
	}

	removed := filterEvents(events, EventAgentRemoved)
	if len(removed) != 0 {
		t.Errorf("expected 0 removed events, got %d", len(removed))
	}
}

func TestDiffSnapshotsAgentAdded(t *testing.T) {
	old := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
		},
		Timestamp: time.Now(),
	}

	new := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
			"%1": {
				Pane:  tmux.Pane{PaneID: "%1"},
				Agent: detector.AgentInfo{Type: detector.AgentAider, Status: detector.StatusThinking},
			},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(old, new)

	added := filterEvents(events, EventAgentAdded)
	if len(added) != 1 {
		t.Fatalf("expected 1 added event, got %d", len(added))
	}
	if added[0].PaneID != "%1" {
		t.Errorf("added pane = %q, want %%1", added[0].PaneID)
	}
	if added[0].Agent.Type != detector.AgentAider {
		t.Errorf("added agent = %q, want aider", added[0].Agent.Type)
	}
}

func TestDiffSnapshotsAgentRemoved(t *testing.T) {
	old := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
			"%1": {
				Pane:  tmux.Pane{PaneID: "%1"},
				Agent: detector.AgentInfo{Type: detector.AgentAider, Status: detector.StatusIdle},
			},
		},
		Timestamp: time.Now(),
	}

	new := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(old, new)

	removed := filterEvents(events, EventAgentRemoved)
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed event, got %d", len(removed))
	}
	if removed[0].PaneID != "%1" {
		t.Errorf("removed pane = %q, want %%1", removed[0].PaneID)
	}
}

func TestDiffSnapshotsStatusChanged(t *testing.T) {
	old := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
		},
		Timestamp: time.Now(),
	}

	new := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusThinking},
			},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(old, new)

	changed := filterEvents(events, EventAgentStatusChanged)
	if len(changed) != 1 {
		t.Fatalf("expected 1 status-changed event, got %d", len(changed))
	}
	if changed[0].Agent.Status != detector.StatusThinking {
		t.Errorf("new status = %q, want %q", changed[0].Agent.Status, detector.StatusThinking)
	}
}

func TestDiffSnapshotsNoChange(t *testing.T) {
	snap := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {
				Pane:  tmux.Pane{PaneID: "%0"},
				Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(snap, snap)
	if len(events) != 0 {
		t.Errorf("expected 0 events for identical snapshots, got %d", len(events))
	}
}

func TestDiffBothNil(t *testing.T) {
	events := diffSnapshots(nil, nil)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestDiffMixedChanges(t *testing.T) {
	old := &Snapshot{
		Agents: map[string]PaneAgent{
			"%0": {Pane: tmux.Pane{PaneID: "%0"}, Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle}},
			"%1": {Pane: tmux.Pane{PaneID: "%1"}, Agent: detector.AgentInfo{Type: detector.AgentAider, Status: detector.StatusThinking}},
			"%2": {Pane: tmux.Pane{PaneID: "%2"}, Agent: detector.AgentInfo{Type: detector.AgentCodex, Status: detector.StatusToolUse}},
		},
		Timestamp: time.Now(),
	}

	new := &Snapshot{
		Agents: map[string]PaneAgent{
			// %0: status changed idle -> thinking
			"%0": {Pane: tmux.Pane{PaneID: "%0"}, Agent: detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusThinking}},
			// %1: removed
			// %2: unchanged
			"%2": {Pane: tmux.Pane{PaneID: "%2"}, Agent: detector.AgentInfo{Type: detector.AgentCodex, Status: detector.StatusToolUse}},
			// %3: added
			"%3": {Pane: tmux.Pane{PaneID: "%3"}, Agent: detector.AgentInfo{Type: detector.AgentGoose, Status: detector.StatusIdle}},
		},
		Timestamp: time.Now(),
	}

	events := diffSnapshots(old, new)

	added := filterEvents(events, EventAgentAdded)
	removed := filterEvents(events, EventAgentRemoved)
	changed := filterEvents(events, EventAgentStatusChanged)

	if len(added) != 1 {
		t.Errorf("expected 1 added, got %d", len(added))
	}
	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(removed))
	}
	if len(changed) != 1 {
		t.Errorf("expected 1 changed, got %d", len(changed))
	}
}

func TestAgentChanged(t *testing.T) {
	tests := []struct {
		name string
		a, b detector.AgentInfo
		want bool
	}{
		{
			name: "identical",
			a:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			b:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			want: false,
		},
		{
			name: "status changed",
			a:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			b:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusThinking},
			want: true,
		},
		{
			name: "type changed",
			a:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle},
			b:    detector.AgentInfo{Type: detector.AgentAider, Status: detector.StatusIdle},
			want: true,
		},
		{
			name: "cwd different but type and status same",
			a:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle, CWD: "/a"},
			b:    detector.AgentInfo{Type: detector.AgentClaudeCode, Status: detector.StatusIdle, CWD: "/b"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agentChanged(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("agentChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func filterEvents(events []Event, kind EventKind) []Event {
	var out []Event
	for _, e := range events {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

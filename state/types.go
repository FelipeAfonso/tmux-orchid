// Package state manages a periodically refreshed view of all AI coding
// agents running across tmux panes, grouped into projects by git root.
package state

import (
	"time"

	"github.com/FelipeAfonso/tmux-orchid/detector"
	"github.com/FelipeAfonso/tmux-orchid/tmux"
)

// PaneAgent pairs a tmux pane with its detected agent information.
type PaneAgent struct {
	// Pane is the tmux pane metadata.
	Pane tmux.Pane
	// Agent is the detected agent info for this pane.
	Agent detector.AgentInfo
}

// Project groups agents that share the same git repository root.
type Project struct {
	// GitRoot is the absolute path to the git repository root.
	// If the agent's CWD is not inside a git repo, this is the CWD itself.
	GitRoot string
	// Name is a short display name derived from the git root directory name.
	Name string
	// Agents lists every detected agent in this project.
	Agents []PaneAgent
}

// Snapshot is a point-in-time view of all detected agents and projects.
type Snapshot struct {
	// Projects is the list of projects, each with its agents.
	Projects []Project
	// Agents is the complete map of pane ID to PaneAgent for quick lookup.
	Agents map[string]PaneAgent
	// Timestamp is when this snapshot was taken.
	Timestamp time.Time
}

// EventKind describes what changed between two snapshots.
type EventKind int

const (
	// EventAgentAdded means a new agent was detected in a pane.
	EventAgentAdded EventKind = iota
	// EventAgentRemoved means an agent is no longer detected in a pane.
	EventAgentRemoved
	// EventAgentStatusChanged means an agent's status changed.
	EventAgentStatusChanged
	// EventSnapshotUpdated means the full snapshot was refreshed.
	EventSnapshotUpdated
)

// Event represents a change between two consecutive snapshots.
type Event struct {
	// Kind describes the type of change.
	Kind EventKind
	// PaneID is the affected pane (empty for EventSnapshotUpdated).
	PaneID string
	// Agent is the current agent info (zero value for EventAgentRemoved).
	Agent detector.AgentInfo
	// Snapshot is the new snapshot (set only for EventSnapshotUpdated).
	Snapshot *Snapshot
}

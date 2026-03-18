package state

import "github.com/FelipeAfonso/tmux-orchid/detector"

// diffSnapshots compares an old and new snapshot, returning a list of
// granular events describing what changed.
func diffSnapshots(oldSnap, newSnap *Snapshot) []Event {
	if oldSnap == nil {
		return diffFromEmpty(newSnap)
	}

	var events []Event

	// Detect removed agents (in old but not in new).
	for paneID, oldPA := range oldSnap.Agents {
		if _, ok := newSnap.Agents[paneID]; !ok {
			events = append(events, Event{
				Kind:   EventAgentRemoved,
				PaneID: paneID,
				Agent:  oldPA.Agent,
			})
		}
	}

	// Detect added agents and status changes.
	for paneID, newPA := range newSnap.Agents {
		oldPA, existed := oldSnap.Agents[paneID]
		if !existed {
			events = append(events, Event{
				Kind:   EventAgentAdded,
				PaneID: paneID,
				Agent:  newPA.Agent,
			})
			continue
		}

		if oldPA.Agent.Status != newPA.Agent.Status {
			events = append(events, Event{
				Kind:   EventAgentStatusChanged,
				PaneID: paneID,
				Agent:  newPA.Agent,
			})
		}
	}

	return events
}

// diffFromEmpty treats all agents in newSnap as newly added.
func diffFromEmpty(newSnap *Snapshot) []Event {
	if newSnap == nil {
		return nil
	}
	events := make([]Event, 0, len(newSnap.Agents))
	for paneID, pa := range newSnap.Agents {
		events = append(events, Event{
			Kind:   EventAgentAdded,
			PaneID: paneID,
			Agent:  pa.Agent,
		})
	}
	return events
}

// agentChanged returns true if the agent info differs in a meaningful way.
func agentChanged(a, b detector.AgentInfo) bool {
	return a.Type != b.Type || a.Status != b.Status
}

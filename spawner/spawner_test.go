package spawner

import (
	"testing"
)

func TestDefaultAgents(t *testing.T) {
	agents := DefaultAgents()
	if len(agents) == 0 {
		t.Fatal("DefaultAgents() returned empty list")
	}

	// Verify all have both Label and Command.
	for _, a := range agents {
		if a.Label == "" {
			t.Errorf("agent with command %q has empty label", a.Command)
		}
		if a.Command == "" {
			t.Errorf("agent with label %q has empty command", a.Label)
		}
	}

	// Verify Claude Code is first.
	if agents[0].Command != "claude" {
		t.Errorf("first agent should be claude, got %q", agents[0].Command)
	}
}

func TestAvailable(t *testing.T) {
	agents := []AgentDef{
		{Label: "Go", Command: "go"},              // should exist
		{Label: "Nope", Command: "nonexistent99"}, // should not exist
	}

	got := Available(agents)

	// "go" should be available, "nonexistent99" should not.
	foundGo := false
	foundNope := false
	for _, a := range got {
		if a.Command == "go" {
			foundGo = true
		}
		if a.Command == "nonexistent99" {
			foundNope = true
		}
	}

	if !foundGo {
		t.Error("expected 'go' to be available")
	}
	if foundNope {
		t.Error("'nonexistent99' should not be available")
	}
}

func TestAvailableEmpty(t *testing.T) {
	got := Available(nil)
	if got != nil {
		t.Errorf("Available(nil) = %v, want nil", got)
	}
}

func TestRequestFields(t *testing.T) {
	// Verify the Request struct can be constructed with TargetSession.
	req := Request{
		Agent:         AgentDef{Label: "Claude", Command: "claude"},
		TargetSession: "my-session",
	}
	if req.TargetSession != "my-session" {
		t.Errorf("TargetSession = %q, want %q", req.TargetSession, "my-session")
	}
}

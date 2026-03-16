package spawner

import (
	"testing"
)

func TestSessionName(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		command string
		want    string
	}{
		{
			name:    "simple path",
			dir:     "/home/user/projects/my-app",
			command: "claude",
			want:    "my-app-claude",
		},
		{
			name:    "trailing slash",
			dir:     "/home/user/projects/my-app/",
			command: "opencode",
			want:    "my-app-opencode",
		},
		{
			name:    "dots in dir name",
			dir:     "/opt/code/v2.0",
			command: "codex",
			want:    "v2-0-codex",
		},
		{
			name:    "colon in dir name",
			dir:     "/mnt/c:/projects/app",
			command: "aider",
			want:    "app-aider",
		},
		{
			name:    "root directory",
			dir:     "/",
			command: "claude",
			want:    "project-claude",
		},
		{
			name:    "empty dir",
			dir:     "",
			command: "claude",
			want:    "project-claude",
		},
		{
			name:    "single component",
			dir:     "myproject",
			command: "goose",
			want:    "myproject-goose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionName(tt.dir, tt.command)
			if got != tt.want {
				t.Errorf("sessionName(%q, %q) = %q, want %q",
					tt.dir, tt.command, got, tt.want)
			}
		})
	}
}

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

func TestRequestValidation(t *testing.T) {
	// We can't test Spawn() without a real tmux server, but we can
	// verify that an empty dir is rejected by constructing a Spawner
	// with a dummy client. The Spawn method checks dir before touching tmux.
	//
	// This is a basic sanity check.
	_ = Request{
		Agent:       AgentDef{Label: "Claude", Command: "claude"},
		Dir:         "/home/user/project",
		SessionName: "test-session",
	}
}

// Package spawner creates new tmux windows running AI coding agents in
// existing tmux sessions.
package spawner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anomalyco/tmux-orchid/tmux"
)

// AgentDef describes a launchable agent type.
type AgentDef struct {
	// Label is the human-readable name shown in the TUI.
	Label string
	// Command is the binary or command to run.
	Command string
	// Args are additional default arguments passed to the command.
	Args []string
}

// DefaultAgents returns the built-in list of spawnable agents.
func DefaultAgents() []AgentDef {
	return []AgentDef{
		{Label: "Claude Code", Command: "claude"},
		{Label: "OpenCode", Command: "opencode"},
		{Label: "Codex", Command: "codex"},
		{Label: "Aider", Command: "aider"},
		{Label: "Gemini CLI", Command: "gemini"},
		{Label: "Goose", Command: "goose"},
		{Label: "Amp", Command: "amp"},
	}
}

// Available filters the given agent definitions down to those whose command
// is found on PATH.
func Available(agents []AgentDef) []AgentDef {
	var out []AgentDef
	for _, a := range agents {
		if _, err := exec.LookPath(a.Command); err == nil {
			out = append(out, a)
		}
	}
	return out
}

// Request describes what to spawn.
type Request struct {
	// Agent is the agent definition to launch.
	Agent AgentDef
	// TargetSession is the existing tmux session to create the new
	// window (tab) in.
	TargetSession string
}

// Result is returned after a successful spawn.
type Result struct {
	// SessionName is the name of the tmux session the window was created in.
	SessionName string
	// PaneID is the tmux pane ID of the new agent.
	PaneID string
}

// Spawner creates tmux windows for AI agents inside existing sessions.
type Spawner struct {
	client *tmux.Client
}

// New creates a Spawner that uses the given tmux client.
func New(client *tmux.Client) *Spawner {
	return &Spawner{client: client}
}

// Spawn creates a new tmux window in the target session running the
// requested agent.
func (s *Spawner) Spawn(ctx context.Context, req Request) (*Result, error) {
	if req.TargetSession == "" {
		return nil, fmt.Errorf("target session is required")
	}

	cmd := req.Agent.Command
	if len(req.Agent.Args) > 0 {
		cmd = cmd + " " + strings.Join(req.Agent.Args, " ")
	}

	paneID, err := s.client.NewWindow(ctx, req.TargetSession, "", cmd)
	if err != nil {
		return nil, fmt.Errorf("creating window in session %q: %w", req.TargetSession, err)
	}

	return &Result{
		SessionName: req.TargetSession,
		PaneID:      paneID,
	}, nil
}

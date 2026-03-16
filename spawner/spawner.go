// Package spawner creates new tmux sessions running AI coding agents in a
// chosen project directory.
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
	// Dir is the working directory (project root) for the new session.
	Dir string
	// SessionName overrides the auto-generated session name. If empty, a
	// name is derived from Dir and Agent.Command.
	SessionName string
}

// Result is returned after a successful spawn.
type Result struct {
	// SessionName is the name of the created tmux session.
	SessionName string
	// PaneID is the tmux pane ID of the new agent.
	PaneID string
}

// Spawner creates tmux sessions for AI agents.
type Spawner struct {
	client *tmux.Client
}

// New creates a Spawner that uses the given tmux client.
func New(client *tmux.Client) *Spawner {
	return &Spawner{client: client}
}

// Spawn creates a new tmux session running the requested agent in the
// specified directory. If a session with the computed name already exists,
// a numeric suffix is appended.
func (s *Spawner) Spawn(ctx context.Context, req Request) (*Result, error) {
	if req.Dir == "" {
		return nil, fmt.Errorf("directory is required")
	}

	name := req.SessionName
	if name == "" {
		name = sessionName(req.Dir, req.Agent.Command)
	}

	// Ensure unique session name.
	name = s.uniqueSessionName(ctx, name)

	cmd := req.Agent.Command
	if len(req.Agent.Args) > 0 {
		cmd = cmd + " " + strings.Join(req.Agent.Args, " ")
	}

	paneID, err := s.newSession(ctx, name, req.Dir, cmd)
	if err != nil {
		return nil, fmt.Errorf("creating session %q: %w", name, err)
	}

	return &Result{
		SessionName: name,
		PaneID:      paneID,
	}, nil
}

// newSession creates a tmux session with the given name, working directory,
// and shell command. Returns the pane ID.
func (s *Spawner) newSession(ctx context.Context, name, dir, cmd string) (string, error) {
	args := []string{
		"new-session",
		"-d",       // detached
		"-s", name, // session name
		"-c", dir, // working directory
		"-P",               // print info
		"-F", "#{pane_id}", // output format
		cmd, // shell command
	}

	out, err := s.run(ctx, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// uniqueSessionName returns name if no session with that name exists,
// otherwise appends -2, -3, etc. until a unique name is found.
func (s *Spawner) uniqueSessionName(ctx context.Context, name string) string {
	if !s.sessionExists(ctx, name) {
		return name
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", name, i)
		if !s.sessionExists(ctx, candidate) {
			return candidate
		}
	}
	return name // give up, let tmux error
}

// sessionExists checks whether a tmux session with the given name exists.
func (s *Spawner) sessionExists(ctx context.Context, name string) bool {
	_, err := s.run(ctx, "has-session", "-t", name)
	return err == nil
}

// run executes a tmux command.
func (s *Spawner) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, s.client.TmuxPath, args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("tmux %s: %w: %s", args[0], err, stderr)
		}
		return "", fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return string(out), nil
}

// sessionName derives a tmux session name from a directory and agent command.
// e.g. "/home/user/projects/my-app" + "claude" -> "my-app-claude"
func sessionName(dir, command string) string {
	base := strings.TrimRight(dir, "/")
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if base == "" {
		base = "project"
	}
	// Sanitise: tmux session names can't contain '.' or ':'
	base = strings.NewReplacer(".", "-", ":", "-").Replace(base)
	command = strings.NewReplacer(".", "-", ":", "-").Replace(command)
	return base + "-" + command
}

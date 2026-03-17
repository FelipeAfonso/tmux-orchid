package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Client executes tmux commands by shelling out to the tmux binary.
type Client struct {
	// TmuxPath is the path to the tmux binary. Defaults to "tmux".
	TmuxPath string
}

// NewClient returns a Client that uses the given tmux binary path.
// If tmuxPath is empty, "tmux" is used (resolved via PATH).
func NewClient(tmuxPath string) *Client {
	if tmuxPath == "" {
		tmuxPath = "tmux"
	}
	return &Client{TmuxPath: tmuxPath}
}

// run executes a tmux subcommand with the given arguments and returns the
// combined stdout output. Stderr is captured and included in any error.
func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.TmuxPath, args...)
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

// Ping checks whether the tmux server is reachable by running `tmux info`.
// It returns nil if the server responds, or an error otherwise.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.run(ctx, "info")
	return err
}

// ListPanes returns all panes across all sessions and windows.
func (c *Client) ListPanes(ctx context.Context) ([]Pane, error) {
	out, err := c.run(ctx, "list-panes", "-a", "-F", paneFormat)
	if err != nil {
		return nil, err
	}
	return ParsePanes(out)
}

// CapturePane captures the visible content of the pane identified by paneID
// (e.g. "%0"). It returns the raw text contents of the pane.
func (c *Client) CapturePane(ctx context.Context, paneID string) (string, error) {
	out, err := c.run(ctx, "capture-pane", "-t", paneID, "-p")
	if err != nil {
		return "", err
	}
	return out, nil
}

// SwitchToPane switches the client to the window and pane identified by
// paneID (e.g. "%0"). This makes the target pane the active pane.
// If the pane lives in a different session, switch-client moves the
// current client to that session first.
func (c *Client) SwitchToPane(ctx context.Context, paneID string) error {
	// switch-client handles the cross-session case; it is a no-op when
	// the pane already belongs to the current session.
	_, _ = c.run(ctx, "switch-client", "-t", paneID)
	_, _ = c.run(ctx, "select-window", "-t", paneID)
	_, err := c.run(ctx, "select-pane", "-t", paneID)
	return err
}

// NewWindow creates a new tmux window running the given shell command.
// It returns the pane ID of the new window's pane.
func (c *Client) NewWindow(ctx context.Context, command string) (string, error) {
	out, err := c.run(ctx, "new-window", "-P", "-F", "#{pane_id}", command)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HasSession reports whether a session with the given name exists.
func (c *Client) HasSession(ctx context.Context, name string) bool {
	_, err := c.run(ctx, "has-session", "-t", name)
	return err == nil
}

// NewSession creates a new detached session with the given name, running
// command in workDir. It returns the pane ID of the initial pane.
func (c *Client) NewSession(ctx context.Context, name, workDir, command string) (string, error) {
	args := []string{
		"new-session",
		"-d",
		"-s", name,
		"-P", "-F", "#{pane_id}",
	}
	if workDir != "" {
		args = append(args, "-c", workDir)
	}
	if command != "" {
		args = append(args, command)
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// SwitchClient switches the current tmux client to the given target
// (session, window, or pane identifier).
func (c *Client) SwitchClient(ctx context.Context, target string) error {
	_, err := c.run(ctx, "switch-client", "-t", target)
	return err
}

// CurrentSession returns the session name of the current tmux client.
func (c *Client) CurrentSession(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "display-message", "-p", "#{session_name}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// BindKey registers a tmux key binding in the given key table. When
// prefix is true, the binding lives in the prefix table (triggered after
// the tmux prefix key).
func (c *Client) BindKey(ctx context.Context, key, command string) error {
	_, err := c.run(ctx, "bind-key", key, "switch-client", "-t", command)
	return err
}

// UnbindKey removes a previously registered key binding.
func (c *Client) UnbindKey(ctx context.Context, key string) error {
	_, err := c.run(ctx, "unbind-key", key)
	return err
}

// KillSession destroys the tmux session with the given name.
func (c *Client) KillSession(ctx context.Context, name string) error {
	_, err := c.run(ctx, "kill-session", "-t", name)
	return err
}

// Executable returns the path to the tmux binary this client uses.
func (c *Client) Executable() string {
	return c.TmuxPath
}

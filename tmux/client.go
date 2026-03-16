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
func (c *Client) SwitchToPane(ctx context.Context, paneID string) error {
	_, err := c.run(ctx, "select-pane", "-t", paneID)
	return err
}

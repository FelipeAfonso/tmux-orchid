package tmux

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// DefaultSessionName is the tmux session name used by the orchid dashboard.
const DefaultSessionName = "orchid"

// EnsureResult describes the outcome of EnsureSession.
type EnsureResult int

const (
	// ResultRunHere means the current process is already in the orchid
	// session and should proceed to run the TUI.
	ResultRunHere EnsureResult = iota
	// ResultRelocated means a new orchid session was created (or already
	// existed) and the client was switched to it. The current process
	// should exit cleanly.
	ResultRelocated
)

// EnsureSession guarantees that the orchid dashboard session exists and
// the current tmux client is attached to it.
//
// If the current process is already running inside the orchid session it
// returns ResultRunHere. Otherwise it creates the session (if needed)
// running the same binary, switches the current client to it, and returns
// ResultRelocated so the caller can exit.
func EnsureSession(ctx context.Context, c *Client, sessionName string) (EnsureResult, error) {
	if sessionName == "" {
		sessionName = DefaultSessionName
	}

	current, err := c.CurrentSession(ctx)
	if err != nil {
		return 0, fmt.Errorf("detecting current session: %w", err)
	}

	// Already inside the orchid session -- run the TUI here.
	if current == sessionName {
		slog.Debug("already in orchid session", "session", sessionName)
		return ResultRunHere, nil
	}

	// Orchid session exists -- just switch to it (handles duplicate
	// invocations).
	if c.HasSession(ctx, sessionName) {
		slog.Info("orchid session already running, switching to it", "session", sessionName)
		if err := c.SwitchClient(ctx, sessionName); err != nil {
			return 0, fmt.Errorf("switching to existing session %q: %w", sessionName, err)
		}
		return ResultRelocated, nil
	}

	// Create a new detached session running ourselves.
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("resolving own executable: %w", err)
	}

	slog.Info("creating orchid session", "session", sessionName, "exe", exe)
	_, err = c.NewSession(ctx, sessionName, "", exe)
	if err != nil {
		return 0, fmt.Errorf("creating session %q: %w", sessionName, err)
	}

	if err := c.SwitchClient(ctx, sessionName); err != nil {
		return 0, fmt.Errorf("switching to new session %q: %w", sessionName, err)
	}

	return ResultRelocated, nil
}

// InstallKeybind registers a tmux keybinding that switches the client
// back to the orchid session. When usePrefix is true the binding lives
// in the prefix table (prefix+key); when false it lives in the root
// table so the key alone triggers the switch.
func InstallKeybind(ctx context.Context, c *Client, key, sessionName string, usePrefix bool) error {
	if key == "" {
		key = "d"
	}
	if sessionName == "" {
		sessionName = DefaultSessionName
	}
	if usePrefix {
		slog.Info("installing tmux keybind", "key", "prefix+"+key, "target", sessionName)
		return c.BindKey(ctx, key, sessionName)
	}
	slog.Info("installing tmux keybind", "key", key, "target", sessionName)
	return c.BindKeyRoot(ctx, key, sessionName)
}

// RemoveKeybind removes the previously installed keybinding. The
// usePrefix parameter must match the value used during installation.
func RemoveKeybind(ctx context.Context, c *Client, key string, usePrefix bool) error {
	if key == "" {
		key = "d"
	}
	if usePrefix {
		slog.Info("removing tmux keybind", "key", "prefix+"+key)
		return c.UnbindKey(ctx, key)
	}
	slog.Info("removing tmux keybind", "key", key)
	return c.UnbindKeyRoot(ctx, key)
}

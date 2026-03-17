// Package main is the entry point for tmux-orchid.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomalyco/tmux-orchid/config"
	"github.com/anomalyco/tmux-orchid/state"
	"github.com/anomalyco/tmux-orchid/tmux"
	"github.com/anomalyco/tmux-orchid/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-orchid: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	restart := flag.Bool("restart", false, "kill the existing orchid session and start fresh")
	flag.BoolVar(restart, "r", false, "shorthand for --restart")
	flag.Parse()

	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	setupLogging(cfg.Log)

	tc := tmux.NewClient(cfg.TmuxPath)

	// Verify tmux is running.
	if err := tc.Ping(context.Background()); err != nil {
		return fmt.Errorf("tmux not reachable (are you inside a tmux session?): %w", err)
	}

	sessionName := cfg.Session.Name
	if sessionName == "" {
		sessionName = tmux.DefaultSessionName
	}
	keybind := cfg.Session.Keybind

	// When --restart is given, kill the existing orchid session so a
	// fresh one is created below. This makes it easy to rebuild and
	// relaunch during development.
	ctx := context.Background()
	if *restart && tc.HasSession(ctx, sessionName) {
		slog.Info("restart requested, killing existing session", "session", sessionName)
		if err := tc.KillSession(ctx, sessionName); err != nil {
			slog.Warn("failed to kill session for restart", "session", sessionName, "error", err)
		}
	}

	// Ensure the orchid session exists and we are inside it.
	// If we are in a different session, this creates/finds the orchid
	// session, switches the client to it, and returns ResultRelocated
	// so we can exit this invocation cleanly.
	result, err := tmux.EnsureSession(ctx, tc, sessionName)
	if err != nil {
		return fmt.Errorf("ensuring orchid session: %w", err)
	}
	if result == tmux.ResultRelocated {
		// The client has been switched to the orchid session where
		// another instance of tmux-orchid is (or will be) running.
		return nil
	}

	// We are inside the orchid session -- run the dashboard.

	// Install the keybind so the user can jump back (e.g. prefix+d).
	if keybind != "" {
		if err := tmux.InstallKeybind(ctx, tc, keybind, sessionName); err != nil {
			slog.Warn("failed to install keybind", "error", err)
			// Non-fatal: the dashboard still works, just no shortcut.
		}
	}

	// Start the state manager in the background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := state.NewManager(tc, cfg.PollInterval.Duration, cfg.SessionFilter)
	go mgr.Run(ctx)

	// Run the TUI.
	model := tui.New(mgr, tc)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running tui: %w", err)
	}

	// Clean up keybind on exit.
	if keybind != "" {
		cleanCtx := context.Background()
		if err := tmux.RemoveKeybind(cleanCtx, tc, keybind); err != nil {
			slog.Warn("failed to remove keybind", "error", err)
		}
	}

	return nil
}

func setupLogging(logCfg config.LogConfig) {
	level := slog.LevelInfo
	switch logCfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	if logCfg.File != "" {
		f, err := os.OpenFile(logCfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot open log file %s: %s\n", logCfg.File, err)
			handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		} else {
			handler = slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
		}
	} else {
		// When running TUI, send logs to /dev/null by default (stderr is the TUI).
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			devNull = os.Stderr
		}
		handler = slog.NewTextHandler(devNull, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))
}

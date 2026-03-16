// Package main is the entry point for tmux-orchid.
package main

import (
	"context"
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

	// Start the state manager in the background.
	mgr := state.NewManager(tc, cfg.PollInterval.Duration, cfg.SessionFilter)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.Run(ctx)

	// Run the TUI.
	model := tui.New(mgr, tc)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running tui: %w", err)
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

// Package config handles loading and validating tmux-orchid configuration
// from TOML files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the top-level configuration for tmux-orchid.
type Config struct {
	// PollInterval controls how often the state manager polls tmux for pane
	// updates. Defaults to 2s.
	PollInterval Duration `toml:"poll_interval"`

	// TmuxPath is the path to the tmux binary. Empty means "tmux" via PATH.
	TmuxPath string `toml:"tmux_path"`

	// SessionFilter, if non-empty, limits scanning to sessions whose names
	// match any entry in this list.
	SessionFilter []string `toml:"session_filter"`

	// Session holds settings for the persistent dashboard session.
	Session SessionConfig `toml:"session"`

	// Theme holds TUI colour and style overrides.
	Theme ThemeConfig `toml:"theme"`

	// Log configures logging behaviour.
	Log LogConfig `toml:"log"`
}

// SessionConfig controls the persistent tmux session that hosts the
// dashboard.
type SessionConfig struct {
	// Name is the tmux session name for the dashboard. Defaults to
	// "orchid".
	Name string `toml:"name"`

	// Keybind is the tmux prefix key that switches back to the dashboard.
	// Defaults to "d". Set to "" to disable automatic keybind
	// installation.
	Keybind string `toml:"keybind"`
}

// ThemeConfig holds TUI appearance settings.
type ThemeConfig struct {
	// ColorScheme selects a named colour palette ("dark", "light", "auto").
	// Defaults to "auto".
	ColorScheme string `toml:"color_scheme"`
}

// LogConfig controls logging output.
type LogConfig struct {
	// Level is the slog log level ("debug", "info", "warn", "error").
	// Defaults to "info".
	Level string `toml:"level"`

	// File is an optional path to write log output to. If empty, logs go
	// to stderr.
	File string `toml:"file"`
}

// Duration wraps time.Duration so it can be decoded from TOML as a string
// like "2s" or "500ms".
type Duration struct {
	time.Duration
}

// UnmarshalText implements encoding.TextUnmarshaler for TOML string decoding.
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	return nil
}

// MarshalText implements encoding.TextMarshaler for TOML string encoding.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// Default returns a Config populated with sensible default values.
func Default() Config {
	return Config{
		PollInterval: Duration{2 * time.Second},
		Session: SessionConfig{
			Name:    "orchid",
			Keybind: "d",
		},
		Theme: ThemeConfig{
			ColorScheme: "auto",
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

// Load reads a TOML config file from path and returns the parsed Config.
// Fields not present in the file retain their default values.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return cfg, nil
}

// LoadOrDefault attempts to load config from the standard locations.
// It tries, in order:
//  1. $XDG_CONFIG_HOME/tmux-orchid/config.toml
//  2. $HOME/.config/tmux-orchid/config.toml
//
// If no file is found, Default() is returned with no error.
func LoadOrDefault() (Config, error) {
	paths := configPaths()
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return Load(p)
		}
	}
	return Default(), nil
}

// validate checks that the config values are within acceptable bounds.
func (c *Config) validate() error {
	if c.PollInterval.Duration < 100*time.Millisecond {
		return fmt.Errorf("poll_interval must be at least 100ms, got %s", c.PollInterval.Duration)
	}
	if c.PollInterval.Duration > 1*time.Minute {
		return fmt.Errorf("poll_interval must be at most 1m, got %s", c.PollInterval.Duration)
	}

	switch c.Theme.ColorScheme {
	case "dark", "light", "auto", "":
		// ok
	default:
		return fmt.Errorf("unknown color_scheme %q", c.Theme.ColorScheme)
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error", "":
		// ok
	default:
		return fmt.Errorf("unknown log level %q", c.Log.Level)
	}

	return nil
}

// configPaths returns the list of config file paths to try, in priority order.
func configPaths() []string {
	var paths []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "tmux-orchid", "config.toml"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "tmux-orchid", "config.toml"))
	}

	return paths
}

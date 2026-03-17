package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.PollInterval.Duration != 2*time.Second {
		t.Errorf("default poll_interval = %v, want 2s", cfg.PollInterval.Duration)
	}
	if cfg.Theme.ColorScheme != "auto" {
		t.Errorf("default color_scheme = %q, want %q", cfg.Theme.ColorScheme, "auto")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("default log level = %q, want %q", cfg.Log.Level, "info")
	}
	if cfg.TmuxPath != "" {
		t.Errorf("default tmux_path = %q, want empty", cfg.TmuxPath)
	}
	if cfg.Session.Name != "orchid" {
		t.Errorf("default session name = %q, want %q", cfg.Session.Name, "orchid")
	}
	if cfg.Session.Keybind != "d" {
		t.Errorf("default session keybind = %q, want %q", cfg.Session.Keybind, "d")
	}
	if !cfg.Session.PrefixEnabled() {
		t.Error("default PrefixEnabled() = false, want true")
	}
}

func TestLoadFullConfig(t *testing.T) {
	content := `
poll_interval = "500ms"
tmux_path = "/usr/local/bin/tmux"
session_filter = ["dev", "agents"]

[session]
name = "dashboard"
keybind = "o"
use_prefix = false

[theme]
color_scheme = "dark"

[log]
level = "debug"
file = "/tmp/orchid.log"
`
	path := writeTemp(t, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.PollInterval.Duration != 500*time.Millisecond {
		t.Errorf("poll_interval = %v, want 500ms", cfg.PollInterval.Duration)
	}
	if cfg.TmuxPath != "/usr/local/bin/tmux" {
		t.Errorf("tmux_path = %q, want %q", cfg.TmuxPath, "/usr/local/bin/tmux")
	}
	if len(cfg.SessionFilter) != 2 || cfg.SessionFilter[0] != "dev" || cfg.SessionFilter[1] != "agents" {
		t.Errorf("session_filter = %v, want [dev agents]", cfg.SessionFilter)
	}
	if cfg.Session.Name != "dashboard" {
		t.Errorf("session name = %q, want %q", cfg.Session.Name, "dashboard")
	}
	if cfg.Session.Keybind != "o" {
		t.Errorf("session keybind = %q, want %q", cfg.Session.Keybind, "o")
	}
	if cfg.Session.PrefixEnabled() {
		t.Error("PrefixEnabled() = true, want false")
	}
	if cfg.Theme.ColorScheme != "dark" {
		t.Errorf("color_scheme = %q, want %q", cfg.Theme.ColorScheme, "dark")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log level = %q, want %q", cfg.Log.Level, "debug")
	}
	if cfg.Log.File != "/tmp/orchid.log" {
		t.Errorf("log file = %q, want %q", cfg.Log.File, "/tmp/orchid.log")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	content := `
poll_interval = "1s"
`
	path := writeTemp(t, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.PollInterval.Duration != 1*time.Second {
		t.Errorf("poll_interval = %v, want 1s", cfg.PollInterval.Duration)
	}
	// Unset fields should have defaults.
	if cfg.Theme.ColorScheme != "auto" {
		t.Errorf("color_scheme = %q, want default %q", cfg.Theme.ColorScheme, "auto")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("log level = %q, want default %q", cfg.Log.Level, "info")
	}
	if cfg.Session.Name != "orchid" {
		t.Errorf("session name = %q, want default %q", cfg.Session.Name, "orchid")
	}
	if cfg.Session.Keybind != "d" {
		t.Errorf("session keybind = %q, want default %q", cfg.Session.Keybind, "d")
	}
	if !cfg.Session.PrefixEnabled() {
		t.Error("PrefixEnabled() = false, want true (default)")
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	path := writeTemp(t, "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	def := Default()
	if cfg.PollInterval.Duration != def.PollInterval.Duration {
		t.Errorf("empty config poll_interval = %v, want default %v",
			cfg.PollInterval.Duration, def.PollInterval.Duration)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "poll too fast",
			content: `poll_interval = "10ms"`,
			wantErr: true,
		},
		{
			name:    "poll too slow",
			content: `poll_interval = "5m"`,
			wantErr: true,
		},
		{
			name:    "invalid color scheme",
			content: "[theme]\ncolor_scheme = \"neon\"",
			wantErr: true,
		},
		{
			name:    "invalid log level",
			content: "[log]\nlevel = \"trace\"",
			wantErr: true,
		},
		{
			name:    "valid minimum poll",
			content: `poll_interval = "100ms"`,
			wantErr: false,
		},
		{
			name:    "valid maximum poll",
			content: `poll_interval = "1m"`,
			wantErr: false,
		},
		{
			name:    "valid color schemes",
			content: "[theme]\ncolor_scheme = \"light\"",
			wantErr: false,
		},
		{
			name:    "valid log level warn",
			content: "[log]\nlevel = \"warn\"",
			wantErr: false,
		},
		{
			name:    "valid log level error",
			content: "[log]\nlevel = \"error\"",
			wantErr: false,
		},
		{
			name:    "invalid toml syntax",
			content: "this is = not [ valid toml",
			wantErr: true,
		},
		{
			name:    "invalid duration format",
			content: `poll_interval = "banana"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.content)
			_, err := Load(path)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoadOrDefaultNoFile(t *testing.T) {
	// Point XDG to a temp dir with no config file.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	cfg, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault() error: %v", err)
	}
	def := Default()
	if cfg.PollInterval.Duration != def.PollInterval.Duration {
		t.Errorf("poll_interval = %v, want default %v",
			cfg.PollInterval.Duration, def.PollInterval.Duration)
	}

	// Verify that a config file was created.
	createdPath := filepath.Join(tmp, "tmux-orchid", "config.toml")
	if _, err := os.Stat(createdPath); err != nil {
		t.Errorf("expected default config file at %s, got error: %v", createdPath, err)
	}
}

func TestLoadOrDefaultCreatesValidConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	// First call creates the file.
	_, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault() error: %v", err)
	}

	// Second call should load the created file successfully.
	cfg, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault() second call error: %v", err)
	}
	def := Default()
	if cfg.PollInterval.Duration != def.PollInterval.Duration {
		t.Errorf("poll_interval = %v, want %v", cfg.PollInterval.Duration, def.PollInterval.Duration)
	}
	if cfg.Session.Name != def.Session.Name {
		t.Errorf("session name = %q, want %q", cfg.Session.Name, def.Session.Name)
	}
	if cfg.Session.Keybind != def.Session.Keybind {
		t.Errorf("session keybind = %q, want %q", cfg.Session.Keybind, def.Session.Keybind)
	}
	if cfg.Session.PrefixEnabled() != def.Session.PrefixEnabled() {
		t.Errorf("PrefixEnabled() = %v, want %v", cfg.Session.PrefixEnabled(), def.Session.PrefixEnabled())
	}
	if cfg.Theme.ColorScheme != def.Theme.ColorScheme {
		t.Errorf("color_scheme = %q, want %q", cfg.Theme.ColorScheme, def.Theme.ColorScheme)
	}
	if cfg.Log.Level != def.Log.Level {
		t.Errorf("log level = %q, want %q", cfg.Log.Level, def.Log.Level)
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "sub", "dir", "config.toml")

	if err := writeDefaultConfig(path); err != nil {
		t.Fatalf("writeDefaultConfig() error: %v", err)
	}

	// File should exist.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error: %v", path, err)
	}
	if info.Size() == 0 {
		t.Error("config file is empty")
	}

	// File should be valid TOML that parses without error.
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() on created config error: %v", err)
	}
	if cfg.PollInterval.Duration != 2*time.Second {
		t.Errorf("poll_interval = %v, want 2s", cfg.PollInterval.Duration)
	}
}

func TestPreferredConfigPathXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	got := preferredConfigPath()
	want := "/custom/xdg/tmux-orchid/config.toml"
	if got != want {
		t.Errorf("preferredConfigPath() = %q, want %q", got, want)
	}
}

func TestPreferredConfigPathHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got := preferredConfigPath()
	if got == "" {
		t.Error("preferredConfigPath() returned empty string")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("preferredConfigPath() = %q, want absolute path", got)
	}
}

func TestLoadOrDefaultWithFile(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "tmux-orchid")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"),
		[]byte(`poll_interval = "750ms"`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := LoadOrDefault()
	if err != nil {
		t.Fatalf("LoadOrDefault() error: %v", err)
	}
	if cfg.PollInterval.Duration != 750*time.Millisecond {
		t.Errorf("poll_interval = %v, want 750ms", cfg.PollInterval.Duration)
	}
}

func TestPrefixEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  SessionConfig
		want bool
	}{
		{
			name: "nil defaults to true",
			cfg:  SessionConfig{UsePrefix: nil},
			want: true,
		},
		{
			name: "explicit true",
			cfg:  SessionConfig{UsePrefix: boolPtr(true)},
			want: true,
		},
		{
			name: "explicit false",
			cfg:  SessionConfig{UsePrefix: boolPtr(false)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.PrefixEnabled()
			if got != tt.want {
				t.Errorf("PrefixEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadUsePrefixExplicitTrue(t *testing.T) {
	content := `
[session]
use_prefix = true
`
	path := writeTemp(t, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Session.PrefixEnabled() {
		t.Error("PrefixEnabled() = false, want true")
	}
}

func TestLoadUsePrefixExplicitFalse(t *testing.T) {
	content := `
[session]
use_prefix = false
`
	path := writeTemp(t, content)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Session.PrefixEnabled() {
		t.Error("PrefixEnabled() = true, want false")
	}
}

func TestDurationMarshalRoundTrip(t *testing.T) {
	d := Duration{3 * time.Second}
	text, err := d.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error: %v", err)
	}

	var d2 Duration
	if err := d2.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText() error: %v", err)
	}
	if d.Duration != d2.Duration {
		t.Errorf("round-trip: got %v, want %v", d2.Duration, d.Duration)
	}
}

// writeTemp writes content to a temp file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

## Project
tmux-orchid — a tmux-native TUI dashboard for AI coding agents.
See AGENTS.md for full spec.

## Commands
- `go build -o bin/tmux-orchid .` — build
- `go test ./... -race` — test
- `go vet ./...` — lint
- `gofumpt -w .` — format

## Rules
- Go 1.23+, use stdlib `log/slog` for logging
- TUI: charmbracelet/bubbletea + lipgloss + bubbles
- Config: BurntSushi/toml
- No CGO, no C bindings
- All tmux interaction via `os/exec` calling the `tmux` binary
- Platform-specific files use `_linux.go` / `_darwin.go` suffixes
- Every exported function needs a doc comment
- Tests use table-driven patterns
- Error messages are lowercase, no punctuation

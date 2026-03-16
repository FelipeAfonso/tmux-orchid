// Package tmux provides a client for interacting with tmux via os/exec.
package tmux

// Pane represents a single tmux pane and its metadata.
type Pane struct {
	// SessionName is the name of the tmux session containing this pane.
	SessionName string
	// WindowIndex is the zero-based index of the window within the session.
	WindowIndex int
	// WindowName is the display name of the window.
	WindowName string
	// PaneIndex is the zero-based index of the pane within the window.
	PaneIndex int
	// PaneID is the tmux-assigned unique pane identifier (e.g. "%0").
	PaneID string
	// PaneWidth is the width of the pane in columns.
	PaneWidth int
	// PaneHeight is the height of the pane in rows.
	PaneHeight int
	// PaneActive is true if this pane is the active pane in its window.
	PaneActive bool
	// PanePID is the PID of the process running in the pane.
	PanePID int
	// CurrentCommand is the command currently running in the pane.
	CurrentCommand string
	// CurrentPath is the current working directory of the pane.
	CurrentPath string
}

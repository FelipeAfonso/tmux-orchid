package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// paneFormat is the tmux format string used with list-panes to produce
// parseable output. Fields are separated by a tab character.
const paneFormat = "#{session_name}\t#{window_index}\t#{window_name}\t#{pane_index}\t#{pane_id}\t#{pane_width}\t#{pane_height}\t#{pane_active}\t#{pane_pid}\t#{pane_current_command}\t#{pane_current_path}"

// paneFieldCount is the expected number of tab-separated fields per line.
const paneFieldCount = 11

// ParsePanes parses the output of `tmux list-panes -a -F <paneFormat>` into
// a slice of Pane values. Each non-empty line is expected to contain exactly
// paneFieldCount tab-separated fields.
func ParsePanes(output string) ([]Pane, error) {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	panes := make([]Pane, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		p, err := parsePaneLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		panes = append(panes, p)
	}

	return panes, nil
}

// parsePaneLine parses a single tab-separated line into a Pane.
func parsePaneLine(line string) (Pane, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != paneFieldCount {
		return Pane{}, fmt.Errorf("expected %d fields, got %d", paneFieldCount, len(fields))
	}

	windowIndex, err := strconv.Atoi(fields[1])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid window_index %q: %w", fields[1], err)
	}

	paneIndex, err := strconv.Atoi(fields[3])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane_index %q: %w", fields[3], err)
	}

	paneWidth, err := strconv.Atoi(fields[5])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane_width %q: %w", fields[5], err)
	}

	paneHeight, err := strconv.Atoi(fields[6])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane_height %q: %w", fields[6], err)
	}

	paneActive := fields[7] == "1"

	panePID, err := strconv.Atoi(fields[8])
	if err != nil {
		return Pane{}, fmt.Errorf("invalid pane_pid %q: %w", fields[8], err)
	}

	return Pane{
		SessionName:    fields[0],
		WindowIndex:    windowIndex,
		WindowName:     fields[2],
		PaneIndex:      paneIndex,
		PaneID:         fields[4],
		PaneWidth:      paneWidth,
		PaneHeight:     paneHeight,
		PaneActive:     paneActive,
		PanePID:        panePID,
		CurrentCommand: fields[9],
		CurrentPath:    fields[10],
	}, nil
}

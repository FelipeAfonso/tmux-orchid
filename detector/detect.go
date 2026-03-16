package detector

import (
	"context"
	"log/slog"

	"github.com/anomalyco/tmux-orchid/tmux"
)

// PaneCapture provides the ability to capture pane content.
// This interface allows testing without a real tmux server.
type PaneCapture interface {
	CapturePane(ctx context.Context, paneID string) (string, error)
}

// Detect inspects a tmux pane and returns information about any AI coding
// agent running in it. It walks the process tree rooted at the pane's PID
// to find a known agent process, then scrapes the pane content to determine
// the agent's current status.
func Detect(ctx context.Context, pane tmux.Pane, capturer PaneCapture) AgentInfo {
	agent := ClassifyProcess(pane.CurrentCommand, "")
	pid := pane.PanePID
	cwd := pane.CurrentPath

	if agent == AgentUnknown {
		proc, found := walkTree(pane.PanePID)
		if found != AgentUnknown && proc != nil {
			agent = found
			pid = proc.PID
			if proc.CWD != "" {
				cwd = proc.CWD
			}
		}
	}

	if agent == AgentUnknown {
		return AgentInfo{
			Type:   AgentUnknown,
			Status: StatusUnknown,
			CWD:    cwd,
			PID:    pid,
		}
	}

	status := StatusUnknown
	content, err := capturer.CapturePane(ctx, pane.PaneID)
	if err != nil {
		slog.Debug("failed to capture pane content", "pane_id", pane.PaneID, "error", err)
	} else {
		status = DetectStatus(agent, content)
	}

	return AgentInfo{
		Type:   agent,
		Status: status,
		CWD:    cwd,
		PID:    pid,
	}
}

// DetectAll inspects all given panes and returns an AgentInfo for each pane
// that has a detected agent. Panes with no detected agent are omitted.
func DetectAll(ctx context.Context, panes []tmux.Pane, capturer PaneCapture) map[string]AgentInfo {
	results := make(map[string]AgentInfo)
	for _, pane := range panes {
		info := Detect(ctx, pane, capturer)
		if info.Type != AgentUnknown {
			results[pane.PaneID] = info
		}
	}
	return results
}

package detector

import (
	"strings"
)

// statusRule maps a pane content pattern to a Status for a given AgentType.
type statusRule struct {
	agent   AgentType
	pattern string
	status  Status
}

// statusRules are evaluated in order; first match wins. Rules are grouped
// by agent, with more specific patterns listed before generic ones.
var statusRules = []statusRule{
	// ── Claude Code ──────────────────────────────────────────────────
	{AgentClaudeCode, "waiting for input", StatusIdle},
	{AgentClaudeCode, "what would you like to do", StatusIdle},
	{AgentClaudeCode, "how can i help", StatusIdle},
	{AgentClaudeCode, "> ", StatusIdle},
	{AgentClaudeCode, "reading file", StatusToolUse},
	{AgentClaudeCode, "writing file", StatusToolUse},
	{AgentClaudeCode, "editing file", StatusToolUse},
	{AgentClaudeCode, "running command", StatusToolUse},
	{AgentClaudeCode, "executing", StatusToolUse},
	{AgentClaudeCode, "searching", StatusToolUse},
	{AgentClaudeCode, "bash", StatusToolUse},
	{AgentClaudeCode, "read(", StatusToolUse},
	{AgentClaudeCode, "edit(", StatusToolUse},
	{AgentClaudeCode, "write(", StatusToolUse},
	{AgentClaudeCode, "glob(", StatusToolUse},
	{AgentClaudeCode, "grep(", StatusToolUse},
	{AgentClaudeCode, "task(", StatusToolUse},
	{AgentClaudeCode, "thinking", StatusThinking},
	{AgentClaudeCode, "generating", StatusThinking},
	{AgentClaudeCode, "error", StatusError},
	{AgentClaudeCode, "failed", StatusError},
	{AgentClaudeCode, "task completed", StatusDone},
	{AgentClaudeCode, "done!", StatusDone},

	// ── Aider ────────────────────────────────────────────────────────
	{AgentAider, "aider>", StatusIdle},
	{AgentAider, "aider> ", StatusIdle},
	{AgentAider, "/run", StatusToolUse},
	{AgentAider, "editing", StatusToolUse},
	{AgentAider, "applying edit", StatusToolUse},
	{AgentAider, "commit ", StatusToolUse},
	{AgentAider, "sending", StatusThinking},
	{AgentAider, "thinking", StatusThinking},
	{AgentAider, "error", StatusError},
	{AgentAider, "done", StatusDone},

	// ── GitHub Copilot ───────────────────────────────────────────────
	{AgentCopilot, "copilot>", StatusIdle},
	{AgentCopilot, "suggestion", StatusThinking},
	{AgentCopilot, "applying", StatusToolUse},
	{AgentCopilot, "error", StatusError},

	// ── OpenAI Codex ─────────────────────────────────────────────────
	{AgentCodex, "codex>", StatusIdle},
	{AgentCodex, "thinking", StatusThinking},
	{AgentCodex, "running", StatusToolUse},
	{AgentCodex, "writing", StatusToolUse},
	{AgentCodex, "error", StatusError},
	{AgentCodex, "complete", StatusDone},

	// ── Gemini CLI ───────────────────────────────────────────────────
	{AgentGeminiCLI, "gemini>", StatusIdle},
	{AgentGeminiCLI, ">>>", StatusIdle},
	{AgentGeminiCLI, "generating", StatusThinking},
	{AgentGeminiCLI, "thinking", StatusThinking},
	{AgentGeminiCLI, "executing", StatusToolUse},
	{AgentGeminiCLI, "error", StatusError},

	// ── Claude-MD ────────────────────────────────────────────────────
	{AgentClaudeMD, "waiting", StatusIdle},
	{AgentClaudeMD, "processing", StatusThinking},
	{AgentClaudeMD, "writing", StatusToolUse},
	{AgentClaudeMD, "error", StatusError},
	{AgentClaudeMD, "done", StatusDone},

	// ── Goose ────────────────────────────────────────────────────────
	{AgentGoose, "goose>", StatusIdle},
	{AgentGoose, "thinking", StatusThinking},
	{AgentGoose, "running", StatusToolUse},
	{AgentGoose, "executing", StatusToolUse},
	{AgentGoose, "error", StatusError},
	{AgentGoose, "complete", StatusDone},

	// ── Amp ──────────────────────────────────────────────────────────
	{AgentAmplitude, "amp>", StatusIdle},
	{AgentAmplitude, "thinking", StatusThinking},
	{AgentAmplitude, "running", StatusToolUse},
	{AgentAmplitude, "error", StatusError},
	{AgentAmplitude, "done", StatusDone},

	// ── Kilo Code ────────────────────────────────────────────────────
	{AgentKilo, "kilo>", StatusIdle},
	{AgentKilo, "thinking", StatusThinking},
	{AgentKilo, "executing", StatusToolUse},
	{AgentKilo, "error", StatusError},
	{AgentKilo, "complete", StatusDone},

	// ── OpenCode ─────────────────────────────────────────────────────
	// "esc interrupt" in the bottom bar means a task is actively running.
	{AgentOpenCode, "esc interrupt", StatusThinking},
	// Task headers like "▣  Plan" or "▣  Build" indicate active work.
	{AgentOpenCode, "writing command", StatusToolUse},
	{AgentOpenCode, "reading file", StatusToolUse},
	{AgentOpenCode, "editing file", StatusToolUse},
	{AgentOpenCode, "applying", StatusToolUse},
	{AgentOpenCode, "running", StatusToolUse},
	{AgentOpenCode, "searching", StatusToolUse},
	{AgentOpenCode, "asked", StatusIdle},
	{AgentOpenCode, "ctrl+p commands", StatusIdle},
	{AgentOpenCode, "error", StatusError},
	{AgentOpenCode, "completed", StatusDone},

	// ── Cursor ───────────────────────────────────────────────────────
	{AgentCursor, "ready", StatusIdle},
	{AgentCursor, "generating", StatusThinking},
	{AgentCursor, "applying", StatusToolUse},
	{AgentCursor, "error", StatusError},

	// ── Windsurf ─────────────────────────────────────────────────────
	{AgentWindsurf, "ready", StatusIdle},
	{AgentWindsurf, "cascade", StatusThinking},
	{AgentWindsurf, "generating", StatusThinking},
	{AgentWindsurf, "writing", StatusToolUse},
	{AgentWindsurf, "error", StatusError},
}

// tailLines is the number of lines from the bottom of pane content to scan.
const tailLines = 30

// DetectStatus infers an agent's current status by scanning the tail of
// captured pane content for known patterns. Only rules matching the given
// agent type are considered. Returns StatusUnknown if no pattern matches.
func DetectStatus(agent AgentType, paneContent string) Status {
	if agent == AgentUnknown || paneContent == "" {
		return StatusUnknown
	}

	tail := lastNLines(paneContent, tailLines)
	lower := strings.ToLower(tail)

	for _, rule := range statusRules {
		if rule.agent != agent {
			continue
		}
		if strings.Contains(lower, rule.pattern) {
			return rule.status
		}
	}

	return StatusUnknown
}

// lastNLines returns the last n non-empty lines of s joined by newlines.
func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")

	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}

	return strings.Join(lines[len(lines)-n:], "\n")
}

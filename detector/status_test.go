package detector

import (
	"strings"
	"testing"
)

func TestDetectStatus(t *testing.T) {
	tests := []struct {
		name        string
		agent       AgentType
		paneContent string
		want        Status
	}{
		// ── Claude Code statuses ────────────────────────────────────
		{
			name:        "claude-code idle prompt",
			agent:       AgentClaudeCode,
			paneContent: "Some output\n> ",
			want:        StatusIdle,
		},
		{
			name:        "claude-code thinking",
			agent:       AgentClaudeCode,
			paneContent: "Thinking about the problem...",
			want:        StatusThinking,
		},
		{
			name:        "claude-code tool use read",
			agent:       AgentClaudeCode,
			paneContent: "Read(file.go)",
			want:        StatusToolUse,
		},
		{
			name:        "claude-code tool use edit",
			agent:       AgentClaudeCode,
			paneContent: "Edit(main.go)",
			want:        StatusToolUse,
		},
		{
			name:        "claude-code tool use bash",
			agent:       AgentClaudeCode,
			paneContent: "Running command: go test ./...\nbash",
			want:        StatusToolUse,
		},
		{
			name:        "claude-code error",
			agent:       AgentClaudeCode,
			paneContent: "Error: something went wrong",
			want:        StatusError,
		},
		{
			name:        "claude-code done",
			agent:       AgentClaudeCode,
			paneContent: "Task completed successfully",
			want:        StatusDone,
		},

		// ── Aider statuses ──────────────────────────────────────────
		{
			name:        "aider idle prompt",
			agent:       AgentAider,
			paneContent: "aider> ",
			want:        StatusIdle,
		},
		{
			name:        "aider thinking",
			agent:       AgentAider,
			paneContent: "Sending request to model...",
			want:        StatusThinking,
		},
		{
			name:        "aider editing",
			agent:       AgentAider,
			paneContent: "Editing file main.go",
			want:        StatusToolUse,
		},

		// ── Codex statuses ──────────────────────────────────────────
		{
			name:        "codex idle",
			agent:       AgentCodex,
			paneContent: "codex> ",
			want:        StatusIdle,
		},
		{
			name:        "codex thinking",
			agent:       AgentCodex,
			paneContent: "Thinking...",
			want:        StatusThinking,
		},

		// ── Gemini CLI statuses ─────────────────────────────────────
		{
			name:        "gemini idle triple arrow",
			agent:       AgentGeminiCLI,
			paneContent: ">>> ",
			want:        StatusIdle,
		},
		{
			name:        "gemini generating",
			agent:       AgentGeminiCLI,
			paneContent: "Generating response...",
			want:        StatusThinking,
		},

		// ── OpenCode statuses ───────────────────────────────────────
		{
			name:        "opencode active task",
			agent:       AgentOpenCode,
			paneContent: "some output\nesc interrupt\n",
			want:        StatusThinking,
		},
		{
			name:        "opencode idle",
			agent:       AgentOpenCode,
			paneContent: "ctrl+p commands\n",
			want:        StatusIdle,
		},

		// ── Cross-agent pattern isolation ───────────────────────────
		{
			name:        "codex prompt does not match claude-code",
			agent:       AgentClaudeCode,
			paneContent: "codex>",
			want:        StatusUnknown,
		},

		// ── Edge cases ──────────────────────────────────────────────
		{
			name:        "empty pane content",
			agent:       AgentClaudeCode,
			paneContent: "",
			want:        StatusUnknown,
		},
		{
			name:        "unknown agent type",
			agent:       AgentUnknown,
			paneContent: "thinking about something",
			want:        StatusUnknown,
		},
		{
			name:        "no matching pattern",
			agent:       AgentClaudeCode,
			paneContent: "some random output that matches nothing",
			want:        StatusUnknown,
		},
		{
			name:        "case insensitive matching",
			agent:       AgentClaudeCode,
			paneContent: "THINKING about the problem",
			want:        StatusThinking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectStatus(tt.agent, tt.paneContent)
			if got != tt.want {
				t.Errorf("DetectStatus(%q, %q) = %q, want %q",
					tt.agent, tt.paneContent, got, tt.want)
			}
		})
	}
}

func TestLastNLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			n:     5,
			want:  "",
		},
		{
			name:  "fewer lines than n",
			input: "line1\nline2\nline3",
			n:     10,
			want:  "line1\nline2\nline3",
		},
		{
			name:  "exactly n lines",
			input: "a\nb\nc",
			n:     3,
			want:  "a\nb\nc",
		},
		{
			name:  "more lines than n",
			input: "a\nb\nc\nd\ne",
			n:     2,
			want:  "d\ne",
		},
		{
			name:  "trailing blank lines stripped",
			input: "a\nb\nc\n\n\n",
			n:     2,
			want:  "b\nc",
		},
		{
			name:  "trailing whitespace-only lines stripped",
			input: "a\nb\nc\n   \n\t\n",
			n:     5,
			want:  "a\nb\nc",
		},
		{
			name:  "single line",
			input: "hello",
			n:     1,
			want:  "hello",
		},
		{
			name:  "n is zero",
			input: "a\nb",
			n:     0,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastNLines(tt.input, tt.n)
			if got != tt.want {
				t.Errorf("lastNLines(%q, %d) = %q, want %q",
					tt.input, tt.n, got, tt.want)
			}
		})
	}
}

func TestDetectStatusUsesOnlyTailLines(t *testing.T) {
	// Build content where the pattern is beyond the tail window.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "nothing interesting here")
	}
	// Put the "thinking" cue at the very top (outside the last 30 lines).
	content := "thinking\n" + strings.Join(lines, "\n")

	got := DetectStatus(AgentClaudeCode, content)
	if got != StatusUnknown {
		t.Errorf("expected StatusUnknown when pattern is outside tail window, got %q", got)
	}
}

package detector

import "testing"

func TestClassifyProcess(t *testing.T) {
	tests := []struct {
		name        string
		processName string
		cmdline     string
		want        AgentType
	}{
		// ── Direct base-name matches ────────────────────────────────
		{
			name:        "claude-code binary",
			processName: "claude-code",
			want:        AgentClaudeCode,
		},
		{
			name:        "claude-md binary",
			processName: "claude-md",
			want:        AgentClaudeMD,
		},
		{
			name:        "claude binary falls back to claude-code",
			processName: "claude",
			want:        AgentClaudeCode,
		},
		{
			name:        "cursor binary",
			processName: "cursor",
			want:        AgentCursor,
		},
		{
			name:        "windsurf binary",
			processName: "windsurf",
			want:        AgentWindsurf,
		},
		{
			name:        "aider binary",
			processName: "aider",
			want:        AgentAider,
		},
		{
			name:        "copilot binary",
			processName: "copilot",
			want:        AgentCopilot,
		},
		{
			name:        "codex binary",
			processName: "codex",
			want:        AgentCodex,
		},
		{
			name:        "gemini binary",
			processName: "gemini",
			want:        AgentGeminiCLI,
		},
		{
			name:        "opencode binary",
			processName: "opencode",
			want:        AgentOpenCode,
		},
		{
			name:        "goose binary",
			processName: "goose",
			want:        AgentGoose,
		},
		{
			name:        "amp binary exact match",
			processName: "amp",
			want:        AgentAmplitude,
		},
		{
			name:        "kilo binary",
			processName: "kilo",
			want:        AgentKilo,
		},

		// ── Case insensitivity ──────────────────────────────────────
		{
			name:        "uppercase CLAUDE-CODE",
			processName: "CLAUDE-CODE",
			want:        AgentClaudeCode,
		},
		{
			name:        "mixed case Aider",
			processName: "Aider",
			want:        AgentAider,
		},

		// ── Full path in process name ───────────────────────────────
		{
			name:        "full path to claude-code",
			processName: "/usr/local/bin/claude-code",
			want:        AgentClaudeCode,
		},
		{
			name:        "full path to aider",
			processName: "/home/user/.local/bin/aider",
			want:        AgentAider,
		},

		// ── Cmdline fallback ────────────────────────────────────────
		{
			name:        "node process with claude-code in cmdline",
			processName: "node",
			cmdline:     "node /usr/lib/claude-code/main.js",
			want:        AgentClaudeCode,
		},
		{
			name:        "python process with aider in cmdline",
			processName: "python3",
			cmdline:     "python3 -m aider --model gpt-4",
			want:        AgentAider,
		},

		// ── amp false-positive prevention ───────────────────────────
		{
			name:        "trampoline should not match amp",
			processName: "trampoline",
			want:        AgentUnknown,
		},
		{
			name:        "samplers should not match amp",
			processName: "samplers",
			want:        AgentUnknown,
		},
		{
			name:        "amp in cmdline should not match (exact pattern)",
			processName: "node",
			cmdline:     "node /usr/lib/amp/server.js",
			want:        AgentUnknown,
		},

		// ── Unknown process ─────────────────────────────────────────
		{
			name:        "bash is not an agent",
			processName: "bash",
			want:        AgentUnknown,
		},
		{
			name:        "vim is not an agent",
			processName: "vim",
			want:        AgentUnknown,
		},
		{
			name:        "empty process name",
			processName: "",
			want:        AgentUnknown,
		},

		// ── Pattern priority ────────────────────────────────────────
		{
			name:        "claude-code takes priority over claude",
			processName: "claude-code",
			want:        AgentClaudeCode,
		},
		{
			name:        "claude-md takes priority over claude",
			processName: "claude-md",
			want:        AgentClaudeMD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyProcess(tt.processName, tt.cmdline)
			if got != tt.want {
				t.Errorf("ClassifyProcess(%q, %q) = %q, want %q",
					tt.processName, tt.cmdline, got, tt.want)
			}
		})
	}
}

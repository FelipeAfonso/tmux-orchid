package detector

import (
	"path/filepath"
	"strings"
)

// agentPatterns maps a process-name substring (or binary base-name) to its
// AgentType. Patterns are checked in order; first match wins.
var agentPatterns = []struct {
	substring string
	agent     AgentType
}{
	{"claude-code", AgentClaudeCode},
	{"claude-md", AgentClaudeMD},
	{"claude", AgentClaudeCode},
	{"cursor", AgentCursor},
	{"windsurf", AgentWindsurf},
	{"aider", AgentAider},
	{"copilot", AgentCopilot},
	{"codex", AgentCodex},
	{"gemini", AgentGeminiCLI},
	{"opencode", AgentOpenCode},
	{"goose", AgentGoose},
	{"amp", AgentAmplitude},
	{"kilo", AgentKilo},
}

// ClassifyProcess determines the AgentType by examining the process name
// and its full command line. It returns AgentUnknown if no match is found.
func ClassifyProcess(processName string, cmdline string) AgentType {
	lower := strings.ToLower(processName)

	// First try the process base name.
	base := filepath.Base(lower)
	if agent := matchPatterns(base); agent != AgentUnknown {
		return agent
	}

	// Fall back to the full command line.
	if cmdline != "" {
		lowerCmd := strings.ToLower(cmdline)
		if agent := matchPatterns(lowerCmd); agent != AgentUnknown {
			return agent
		}
	}

	return AgentUnknown
}

// matchPatterns checks a string against the known agent patterns.
func matchPatterns(s string) AgentType {
	for _, p := range agentPatterns {
		if strings.Contains(s, p.substring) {
			return p.agent
		}
	}
	return AgentUnknown
}

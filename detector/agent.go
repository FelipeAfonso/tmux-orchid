package detector

import (
	"path/filepath"
	"strings"
)

// agentPattern maps a process-name pattern to its AgentType. When exact is
// true the pattern must equal the entire string being matched (used for
// short, ambiguous names like "amp" that would otherwise false-positive on
// words like "trampoline").
type agentPattern struct {
	substring string
	agent     AgentType
	exact     bool
}

// agentPatterns are checked in order; first match wins.
var agentPatterns = []agentPattern{
	{"claude-code", AgentClaudeCode, false},
	{"claude-md", AgentClaudeMD, false},
	{"claude", AgentClaudeCode, false},
	{"cursor", AgentCursor, false},
	{"windsurf", AgentWindsurf, false},
	{"aider", AgentAider, false},
	{"copilot", AgentCopilot, false},
	{"codex", AgentCodex, false},
	{"gemini", AgentGeminiCLI, false},
	{"opencode", AgentOpenCode, false},
	{"goose", AgentGoose, false},
	{"amp", AgentAmplitude, true},
	{"kilo", AgentKilo, false},
}

// ClassifyProcess determines the AgentType by examining the process name
// and its full command line. It returns AgentUnknown if no match is found.
func ClassifyProcess(processName string, cmdline string) AgentType {
	lower := strings.ToLower(processName)

	// First try the process base name (exact match for short patterns).
	base := filepath.Base(lower)
	if agent := matchPatterns(base, true); agent != AgentUnknown {
		return agent
	}

	// Fall back to the full command line (substring match only for
	// non-exact patterns).
	if cmdline != "" {
		lowerCmd := strings.ToLower(cmdline)
		if agent := matchPatterns(lowerCmd, false); agent != AgentUnknown {
			return agent
		}
	}

	return AgentUnknown
}

// matchPatterns checks a string against the known agent patterns. When
// matchBase is true the string is treated as a binary base name: exact
// patterns require an equality match, while substring patterns use
// strings.Contains. When matchBase is false (e.g. for full command
// lines), exact patterns are skipped entirely to avoid false positives.
func matchPatterns(s string, matchBase bool) AgentType {
	for _, p := range agentPatterns {
		if p.exact {
			// Exact patterns only match against the base name.
			if matchBase && s == p.substring {
				return p.agent
			}
		} else if strings.Contains(s, p.substring) {
			return p.agent
		}
	}
	return AgentUnknown
}

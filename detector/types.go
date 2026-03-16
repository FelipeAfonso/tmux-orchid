// Package detector identifies AI coding agents running in tmux panes
// by inspecting process trees and scraping pane content for status cues.
package detector

// AgentType identifies a known AI coding agent.
type AgentType string

const (
	// AgentUnknown means the process could not be identified as a known agent.
	AgentUnknown AgentType = ""
	// AgentClaudeCode is Anthropic's Claude Code CLI.
	AgentClaudeCode AgentType = "claude-code"
	// AgentCursor is the Cursor AI editor.
	AgentCursor AgentType = "cursor"
	// AgentWindsurf is the Windsurf (Codeium) AI editor.
	AgentWindsurf AgentType = "windsurf"
	// AgentAider is the aider CLI coding assistant.
	AgentAider AgentType = "aider"
	// AgentCopilot is GitHub Copilot in the CLI.
	AgentCopilot AgentType = "copilot"
	// AgentCodex is the OpenAI Codex CLI agent.
	AgentCodex AgentType = "codex"
	// AgentGeminiCLI is Google's Gemini CLI agent.
	AgentGeminiCLI AgentType = "gemini-cli"
	// AgentClaudeMD is the Claude-MD markdown agent.
	AgentClaudeMD AgentType = "claude-md"
	// AgentGoose is the Goose AI agent.
	AgentGoose AgentType = "goose"
	// AgentAmplitude is the Amp AI agent.
	AgentAmplitude AgentType = "amp"
	// AgentKilo is the Kilo Code CLI.
	AgentKilo AgentType = "kilo-code"
	// AgentOpenCode is the OpenCode AI coding agent.
	AgentOpenCode AgentType = "opencode"
)

// Status represents the current activity state of a detected agent.
type Status string

const (
	// StatusUnknown means the status could not be determined.
	StatusUnknown Status = "unknown"
	// StatusIdle means the agent is waiting for user input.
	StatusIdle Status = "idle"
	// StatusThinking means the agent is processing / generating a response.
	StatusThinking Status = "thinking"
	// StatusToolUse means the agent is executing a tool (file edit, shell, etc.).
	StatusToolUse Status = "tool_use"
	// StatusError means the agent has encountered an error.
	StatusError Status = "error"
	// StatusDone means the agent has finished its task.
	StatusDone Status = "done"
)

// AgentInfo holds everything we know about an agent running in a tmux pane.
type AgentInfo struct {
	// Type is the detected agent type.
	Type AgentType
	// Status is the agent's current activity state.
	Status Status
	// CWD is the working directory of the agent process, if known.
	CWD string
	// PID is the PID of the agent process (or the shell if we couldn't
	// walk further).
	PID int
}

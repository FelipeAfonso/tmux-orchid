package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/anomalyco/tmux-orchid/state"
)

// sidebarWidth is the fixed width of the sidebar panel.
const sidebarWidth = 36

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	// Spawn dialogs are overlays.
	if m.mode == modeSpawnAgent {
		return m.viewSpawnAgentDialog()
	}
	if m.mode == modeSpawnSession {
		return m.viewSpawnSessionDialog()
	}

	sidebar := m.viewSidebar()
	detail := m.viewDetail()
	statusBar := m.viewStatusBar()

	// Layout: sidebar | detail, status bar at bottom.
	statusBarHeight := 1
	contentHeight := m.height - statusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	sw := sidebarWidth
	if sw > m.width/2 {
		sw = m.width / 2
	}
	detailWidth := m.width - sw - 2 // 2 for borders
	if detailWidth < 10 {
		detailWidth = 10
	}

	sidebarRendered := sidebarActiveStyle.
		Width(sw).
		Height(contentHeight - 2). // account for border
		Render(sidebar)

	// Use minimal-padding pane style when showing live terminal content,
	// fall back to the padded detail style when no agent is selected.
	dStyle := paneViewStyle
	if m.selectedAgent() == nil {
		dStyle = detailStyle
	}
	detailRendered := dStyle.
		Width(detailWidth).
		Height(contentHeight - 2).
		Render(detail)

	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebarRendered, detailRendered)

	statusBarRendered := statusBarStyle.
		Width(m.width).
		Render(statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, content, statusBarRendered)
}

// viewSidebar renders the project/agent list.
func (m Model) viewSidebar() string {
	if len(m.items) == 0 {
		if m.snapshot == nil {
			return dimStyle.Render("scanning...")
		}
		if m.filterText != "" {
			return dimStyle.Render("no matches")
		}
		return dimStyle.Render("no agents detected")
	}

	var b strings.Builder
	for i, item := range m.items {
		if item.isProject {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(projectHeaderStyle.Render(item.project.Name))
			b.WriteString("\n")
		} else {
			line := m.renderAgentLine(item.agent, i == m.cursor)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderAgentLine renders a single agent row for the sidebar.
func (m Model) renderAgentLine(pa *state.PaneAgent, selected bool) string {
	status := string(pa.Agent.Status)
	icon := statusIcon(status)
	stStyle := statusStyle(status)

	agentName := string(pa.Agent.Type)
	session := pa.Pane.SessionName
	label := fmt.Sprintf(" %s %s", stStyle.Render(icon), agentName)

	if session != "" {
		label += dimStyle.Render(" [" + session + "]")
	}

	if selected {
		return selectedItemStyle.Render(">" + label)
	}
	return normalItemStyle.Render(" " + label)
}

// viewDetail renders the detail panel for the selected agent.
// It shows a compact header (agent type + status) followed by
// the live captured pane content with ANSI colours preserved.
func (m Model) viewDetail() string {
	pa := m.selectedAgent()
	if pa == nil {
		return dimStyle.Render("select an agent to view details")
	}

	// Compact header: "claude-code  * thinking  [session]"
	status := string(pa.Agent.Status)
	stStyle := statusStyle(status)
	header := paneHeaderStyle.Render(string(pa.Agent.Type)) +
		"  " + stStyle.Render(statusIcon(status)+" "+status) +
		"  " + dimStyle.Render("["+pa.Pane.SessionName+"]")

	if m.paneContent == "" {
		return header + "\n\n" + dimStyle.Render("capturing pane...")
	}

	return header + "\n" + m.formatPaneContent()
}

// formatPaneContent clips the captured ANSI pane content to fit the
// detail panel dimensions.
func (m Model) formatPaneContent() string {
	// Available width: total minus sidebar, borders, and minimal padding.
	sw := sidebarWidth
	if sw > m.width/2 {
		sw = m.width / 2
	}
	// 2 for sidebar border, 2 for detail border.
	availWidth := m.width - sw - 4
	if availWidth < 10 {
		availWidth = 10
	}

	// Available height: total minus status bar (1), detail border (2),
	// and header line (1).
	availHeight := m.height - 1 - 2 - 1
	if availHeight < 1 {
		availHeight = 1
	}

	content := strings.TrimRight(m.paneContent, "\n")
	lines := strings.Split(content, "\n")

	// Cap the number of lines to the available height.
	if len(lines) > availHeight {
		lines = lines[:availHeight]
	}

	// Truncate each line to the available width, preserving ANSI escapes.
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, availWidth, "")
	}

	return strings.Join(lines, "\n")
}

// viewStatusBar renders the bottom status bar.
func (m Model) viewStatusBar() string {
	left := ""
	if m.mode == modeFilter {
		left = filterPromptStyle.Render("/") + m.filterText + dimStyle.Render("|")
	} else if m.filterText != "" {
		left = dimStyle.Render("filter: ") + m.filterText
	}

	agentCount := 0
	projectCount := 0
	if m.snapshot != nil {
		agentCount = len(m.snapshot.Agents)
		projectCount = len(m.snapshot.Projects)
	}

	right := dimStyle.Render(fmt.Sprintf(
		"%d agents  %d projects", agentCount, projectCount,
	))

	help := dimStyle.Render("j/k:nav  enter:go to  /:filter  n:new  prefix+d:back  q:quit")

	if left != "" {
		return left + "  " + help + "  " + right
	}
	return help + "  " + right
}

// viewSpawnAgentDialog renders step 1: agent selection overlay.
func (m Model) viewSpawnAgentDialog() string {
	var b strings.Builder

	b.WriteString(dialogTitleStyle.Render("Spawn New Agent"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Select an agent to launch"))
	b.WriteString("\n\n")

	for i, agent := range m.spawnAgents {
		prefix := "  "
		style := dialogNormalStyle
		if i == m.spawnCursor {
			prefix = "> "
			style = dialogSelectedStyle
		}
		b.WriteString(style.Render(prefix + agent.Label))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter:select  esc:cancel"))

	dialog := dialogStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// viewSpawnSessionDialog renders step 2: tmux session selection overlay.
func (m Model) viewSpawnSessionDialog() string {
	var b strings.Builder

	agentLabel := ""
	if m.spawnPicked != nil {
		agentLabel = m.spawnPicked.Label
	}

	b.WriteString(dialogTitleStyle.Render("Choose Session"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("spawn " + agentLabel + " in"))
	b.WriteString("\n\n")

	for i, session := range m.spawnSessions {
		prefix := "  "
		style := dialogNormalStyle
		if i == m.spawnSessionIdx {
			prefix = "> "
			style = dialogSelectedStyle
		}
		b.WriteString(style.Render(prefix + session))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter:spawn  esc:back  q:cancel"))

	dialog := dialogStyle.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

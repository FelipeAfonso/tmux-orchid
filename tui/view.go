package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

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

	// Spawn dialog is an overlay.
	if m.mode == modeSpawn {
		return m.viewSpawnDialog()
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

	detailRendered := detailStyle.
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
func (m Model) viewDetail() string {
	pa := m.selectedAgent()
	if pa == nil {
		return dimStyle.Render("select an agent to view details")
	}

	var b strings.Builder

	// Title.
	agentTitle := titleStyle.Render(string(pa.Agent.Type))
	b.WriteString(agentTitle)
	b.WriteString("\n\n")

	// Status.
	status := string(pa.Agent.Status)
	stStyle := statusStyle(status)
	b.WriteString(detailRow("Status", stStyle.Render(statusIcon(status)+" "+status)))
	b.WriteString(detailRow("PID", fmt.Sprintf("%d", pa.Agent.PID)))
	b.WriteString(detailRow("CWD", pa.Agent.CWD))
	b.WriteString(detailRow("Session", pa.Pane.SessionName))
	b.WriteString(detailRow("Window", fmt.Sprintf("%d:%s", pa.Pane.WindowIndex, pa.Pane.WindowName)))
	b.WriteString(detailRow("Pane", fmt.Sprintf("%s (idx %d)", pa.Pane.PaneID, pa.Pane.PaneIndex)))
	b.WriteString(detailRow("Size", fmt.Sprintf("%dx%d", pa.Pane.PaneWidth, pa.Pane.PaneHeight)))
	b.WriteString(detailRow("Command", pa.Pane.CurrentCommand))

	if pa.Pane.PaneActive {
		b.WriteString(detailRow("Active", "yes"))
	}

	return b.String()
}

// detailRow renders a label: value pair for the detail panel.
func detailRow(label, value string) string {
	return detailLabelStyle.Render(label+":") + " " + detailValueStyle.Render(value) + "\n"
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

	help := dimStyle.Render("j/k:nav  enter:switch  /:filter  n:new  q:quit")

	// Build: left | help | right
	if left != "" {
		return left + "  " + help + "  " + right
	}
	return help + "  " + right
}

// viewSpawnDialog renders the spawn-new-agent overlay.
func (m Model) viewSpawnDialog() string {
	var b strings.Builder

	b.WriteString(dialogTitleStyle.Render("Spawn New Agent"))
	b.WriteString("\n\n")

	for i, opt := range m.spawnOptions {
		prefix := "  "
		style := dialogNormalStyle
		if i == m.spawnCursor {
			prefix = "> "
			style = dialogSelectedStyle
		}
		b.WriteString(style.Render(prefix + opt.label))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter:select  esc:cancel"))

	dialog := dialogStyle.Render(b.String())

	// Center the dialog on screen.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

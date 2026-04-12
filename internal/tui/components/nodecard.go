package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderNodeGrid renders all nodes as a grid of condensed cards.
func RenderNodeGrid(nodes []model.NodeStatus, selectedIdx int, width int, displayNames map[string]string, expanded bool) string {
	if len(nodes) == 0 {
		return ""
	}
	minCardWidth := 40
	numCols := width / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	if numCols > len(nodes) {
		numCols = len(nodes)
	}
	// Subtract 2 for lipgloss rounded border chars (│ on each side, outside Width)
	cardWidth := width/numCols - 2
	if cardWidth < minCardWidth-2 {
		cardWidth = minCardWidth - 2
	}

	var cards []string
	for i, node := range nodes {
		name := node.Hostname
		if dn, ok := displayNames[node.Hostname]; ok {
			name = dn
		}
		cards = append(cards, renderCondensedCard(node, cardWidth, i == selectedIdx, name, expanded))
	}

	// Arrange in rows
	var rows []string
	for i := 0; i < len(cards); i += numCols {
		end := i + numCols
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

// renderCondensedCard renders a compact node card with optional process lines.
func renderCondensedCard(node model.NodeStatus, width int, selected bool, displayName string, expanded bool) string {
	innerWidth := width - 4 // border + padding
	barWidth := innerWidth - 12
	if barWidth < 10 {
		barWidth = 10
	}

	borderColor := ColorBorder

	var titleIcon string
	var contentLines []string

	if !node.IsOnline() {
		titleIcon = "✗"
		borderColor = ColorRed

		errMsg := "Offline"
		if node.Error != nil && len(*node.Error) > 0 {
			errMsg = *node.Error
			if len(errMsg) > innerWidth-4 {
				errMsg = errMsg[:innerWidth-4]
			}
		}
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorRed).Render(" ⚠ "+errMsg),
			"",
			"",
		)
	} else if len(node.GPUs) == 0 {
		titleIcon = "?"
		borderColor = ColorYellow
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorDim).Render(" No GPUs detected"),
			"",
			"",
		)
	} else {
		avgUtil := node.AvgUtilization()
		titleIcon = NodeStatusIcon(avgUtil)
		borderColor = NodeBorderColor(avgUtil)

		// Line 1: Aggregate utilization bar
		utilBar := RenderGradientBar(avgUtil, barWidth, UtilGradient)
		utilPct := lipgloss.NewStyle().Bold(true).Foreground(UtilColor(avgUtil)).Render(fmt.Sprintf("%3.0f%%", avgUtil))
		utilLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("Util ")
		contentLines = append(contentLines, utilLabel+utilBar+" "+utilPct)

		// Line 2: Aggregate memory bar
		memPct := 0.0
		if node.TotalMemory() > 0 {
			memPct = float64(node.TotalMemoryUsed()) / float64(node.TotalMemory()) * 100
		}
		memBar := RenderGradientBar(memPct, barWidth, MemGradient)
		memVal := lipgloss.NewStyle().Bold(true).Foreground(MemColor(memPct)).Render(model.FormatMemory(node.TotalMemoryUsed()))
		memLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("Mem  ")
		contentLines = append(contentLines, memLabel+memBar+" "+memVal)

		// Line 3: GPU heatmap + model + idle indicator
		heatmap := RenderGPUHeatmap(node.GPUs)
		gpuLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("GPU  ")
		modelStr := lipgloss.NewStyle().Foreground(ColorDim).Render(" " + node.GPUModelSummary())

		users := node.ActiveUsers()
		if len(users) == 0 {
			idleStr := lipgloss.NewStyle().Foreground(ColorBorder).Italic(true).Render("  idle")
			contentLines = append(contentLines, gpuLabel+heatmap+modelStr+idleStr)
		} else {
			contentLines = append(contentLines, gpuLabel+heatmap+modelStr)
		}

		// Process lines (only when expanded and there are processes)
		if expanded && len(users) > 0 {
			procLines := renderCardProcesses(node, innerWidth)
			contentLines = append(contentLines, procLines...)
		}
	}

	content := strings.Join(contentLines, "\n")

	// Build card with selection override
	titleColor := borderColor
	if selected {
		borderColor = ColorSelection
		titleColor = ColorSelection
	}

	titleText := lipgloss.NewStyle().Bold(true).Foreground(titleColor).Render(titleIcon + " " + displayName)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	rendered := style.Render(content)

	// Replace top border with embedded title
	lines := strings.Split(rendered, "\n")
	if len(lines) > 1 {
		// Measure actual body line width to ensure title alignment
		bodyWidth := lipgloss.Width(lines[1])
		lines[0] = embedTitle(lines[0], titleText, borderColor, bodyWidth)
	}

	return strings.Join(lines, "\n")
}

// renderCardProcesses renders per-user process lines for a card.
func renderCardProcesses(node model.NodeStatus, innerWidth int) []string {
	type userAgg struct {
		User string
		GPUs []int
		Mem  int
		Cmd  string
	}

	aggMap := make(map[string]*userAgg)
	var userOrder []string
	for _, g := range node.GPUs {
		for _, p := range g.Processes {
			if a, ok := aggMap[p.User]; ok {
				a.GPUs = appendUniqueInt(a.GPUs, p.GPUIndex)
				a.Mem += p.MemoryMiB
			} else {
				aggMap[p.User] = &userAgg{
					User: p.User,
					GPUs: []int{p.GPUIndex},
					Mem:  p.MemoryMiB,
					Cmd:  p.Command,
				}
				userOrder = append(userOrder, p.User)
			}
		}
	}

	var lines []string
	for _, user := range userOrder {
		a := aggMap[user]
		sortInts(a.GPUs)
		gpuStr := formatGPURange(a.GPUs)

		// Command: basename only, truncate
		cmd := baseName(a.Cmd)
		maxCmd := innerWidth - 24
		if maxCmd < 6 {
			maxCmd = 6
		}
		if len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-2] + ".."
		}

		line := lipgloss.NewStyle().Foreground(ColorAccent).Render(fmt.Sprintf("%-5s", truncStr(user, 5))) + " " +
			lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("%-3s", gpuStr)) + " " +
			lipgloss.NewStyle().Foreground(ColorFg).Render(fmt.Sprintf("%4s", model.FormatMemory(a.Mem))) + " " +
			lipgloss.NewStyle().Foreground(ColorDim).Render(cmd)
		lines = append(lines, line)
	}
	return lines
}

// embedTitle replaces the top border line with ╭─┤ title ├───╮.
func embedTitle(topLine, title string, borderColor lipgloss.Color, targetWidth int) string {
	bc := func(s string) string {
		return lipgloss.NewStyle().Foreground(borderColor).Render(s)
	}

	prefix := bc("╭─┤ ")
	suffix := bc(" ├")

	titleRendered := prefix + title + suffix
	titleVisualWidth := lipgloss.Width(titleRendered)

	// Fill remaining width, accounting for closing ╮
	remaining := targetWidth - titleVisualWidth - 1
	if remaining < 0 {
		remaining = 0
	}

	return titleRendered + bc(strings.Repeat("─", remaining)) + bc("╮")
}

// Helper functions

func appendUniqueInt(slice []int, val int) []int {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func baseName(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}
	exe := parts[0]
	idx := strings.LastIndex(exe, "/")
	if idx >= 0 {
		exe = exe[idx+1:]
	}
	parts[0] = exe
	return strings.Join(parts, " ")
}

// formatGPURange formats GPU indices compactly: [0,1,2,3] -> "0-3", [0,2,5] -> "0,2,5"
func formatGPURange(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return fmt.Sprintf("%d", ids[0])
	}
	contiguous := true
	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[i-1]+1 {
			contiguous = false
			break
		}
	}
	if contiguous {
		return fmt.Sprintf("%d-%d", ids[0], ids[len(ids)-1])
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ",")
}

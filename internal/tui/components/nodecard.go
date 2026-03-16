package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderNodeGrid renders all nodes as a grid of condensed cards.
func RenderNodeGrid(nodes []model.NodeStatus, selectedIdx int, width int) string {
	minCardWidth := 40
	numCols := width / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	if numCols > len(nodes) && len(nodes) > 0 {
		numCols = len(nodes)
	}
	cardWidth := width/numCols - 1
	if cardWidth < minCardWidth {
		cardWidth = minCardWidth
	}

	var cards []string
	for i, node := range nodes {
		cards = append(cards, renderCondensedCard(node, cardWidth, i == selectedIdx))
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

// renderCondensedCard renders a compact node card (4 lines content).
func renderCondensedCard(node model.NodeStatus, width int, selected bool) string {
	innerWidth := width - 4 // border + padding
	barWidth := innerWidth - 12 // label + pct
	if barWidth < 10 {
		barWidth = 10
	}

	borderColor := ColorBorder

	// Build title
	var titleIcon, titleName string
	var contentLines []string

	if !node.IsOnline() {
		titleIcon = "✗"
		titleName = node.Hostname
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
		titleName = node.Hostname
		borderColor = ColorYellow
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorDim).Render(" No GPUs detected"),
			"",
			"",
		)
	} else {
		avgUtil := node.AvgUtilization()
		titleIcon = NodeStatusIcon(avgUtil)
		titleName = node.Hostname
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
		memVal := lipgloss.NewStyle().Bold(true).Foreground(MemColor(memPct)).Render(
			fmt.Sprintf("%s", model.FormatMemory(node.TotalMemoryUsed())),
		)
		memLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("Mem  ")
		contentLines = append(contentLines, memLabel+memBar+" "+memVal)

		// Line 3: GPU heatmap + model + users
		heatmap := RenderGPUHeatmap(node.GPUs)
		gpuLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("GPU  ")
		modelStr := lipgloss.NewStyle().Foreground(ColorDim).Render(" " + node.GPUModelSummary())

		userStr := ""
		users := node.ActiveUsers()
		if len(users) > 0 {
			joined := strings.Join(users, ",")
			maxLen := innerWidth - len(node.GPUModelSummary()) - len(node.GPUs) - 10
			if maxLen < 0 {
				maxLen = 0
			}
			if len(joined) > maxLen {
				if maxLen > 3 {
					joined = joined[:maxLen-2] + ".."
				} else {
					joined = ""
				}
			}
			if joined != "" {
				userStr = lipgloss.NewStyle().Foreground(ColorAccent).Render(" " + joined)
			}
		}
		contentLines = append(contentLines, gpuLabel+heatmap+modelStr+userStr)
	}

	content := strings.Join(contentLines, "\n")

	// Build the card with btop-style title in border
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
	if selected {
		borderColor = ColorSelection
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSelection)
	}
	title := titleStyle.Render(titleIcon + " " + titleName)

	// Card border
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	// Render with title embedded
	rendered := style.Render(content)

	// Replace top border center with title
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		lines[0] = embedTitle(lines[0], title, borderColor, width)
	}

	return strings.Join(lines, "\n")
}

// embedTitle replaces the center of a top border line with a title.
// Input: "╭────────────────────╮"
// Output: "╭─┤ hostname ⚡ ├────╮"
func embedTitle(topLine, title string, borderColor lipgloss.Color, width int) string {
	left := lipgloss.NewStyle().Foreground(borderColor).Render("╭─┤ ")
	right := lipgloss.NewStyle().Foreground(borderColor).Render(" ├")

	// Calculate remaining border to fill
	titleRendered := left + title + right
	titleWidth := lipgloss.Width(titleRendered)
	remaining := width - titleWidth - 1 // -1 for the closing ╮
	if remaining < 0 {
		remaining = 0
	}

	fill := lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", remaining))
	close := lipgloss.NewStyle().Foreground(borderColor).Render("╮")

	return titleRendered + fill + close
}

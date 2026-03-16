package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderHeader renders the header line content (for embedding in outer frame border).
func RenderHeader(nodes []model.NodeStatus, interval float64, width int) string {
	now := time.Now().Format("15:04:05")

	totalGPUs := 0
	onlineNodes := 0
	offlineNodes := 0
	totalUtil := 0
	totalMemUsed := 0
	totalMem := 0

	for _, n := range nodes {
		if n.IsOnline() {
			onlineNodes++
			totalGPUs += n.TotalGPUs()
			totalMemUsed += n.TotalMemoryUsed()
			totalMem += n.TotalMemory()
			for _, g := range n.GPUs {
				totalUtil += g.Utilization
			}
		} else {
			offlineNodes++
		}
	}

	avgUtil := 0.0
	if totalGPUs > 0 {
		avgUtil = float64(totalUtil) / float64(totalGPUs)
	}
	avgMemPct := 0.0
	if totalMem > 0 {
		avgMemPct = float64(totalMemUsed) / float64(totalMem) * 100
	}

	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render(" │ ")

	// Left side: title + stats
	title := lipgloss.NewStyle().Bold(true).Foreground(ColorFg).Render("GPU Cluster Monitor")

	nodeStr := lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("%d", onlineNodes))
	if offlineNodes > 0 {
		nodeStr += lipgloss.NewStyle().Foreground(ColorRed).Render(fmt.Sprintf("/%d", offlineNodes))
	}
	nodeStr += lipgloss.NewStyle().Foreground(ColorDim).Render(" nodes")

	gpuStr := lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("%d GPUs", totalGPUs))
	utilStr := lipgloss.NewStyle().Bold(true).Foreground(UtilColor(avgUtil)).Render(fmt.Sprintf("⚡%.0f%%", avgUtil))
	memStr := lipgloss.NewStyle().Bold(true).Foreground(MemColor(avgMemPct)).Render(
		fmt.Sprintf("💾%s", model.FormatMemory(totalMemUsed)),
	)

	leftParts := title + sep + nodeStr + sep + gpuStr + sep + utilStr + sep + memStr

	// Right side: keybinding hints + time
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	descStyle := lipgloss.NewStyle().Foreground(ColorDim)

	hints := keyStyle.Render("s") + descStyle.Render(":sort ") +
		keyStyle.Render("/") + descStyle.Render(":search ") +
		keyStyle.Render("?") + descStyle.Render(":help")

	timeStr := lipgloss.NewStyle().Foreground(ColorDim).Render(now)

	rightParts := hints + sep + timeStr

	// Fill middle with spaces
	leftWidth := lipgloss.Width(leftParts)
	rightWidth := lipgloss.Width(rightParts)
	gap := width - leftWidth - rightWidth - 4 // padding
	if gap < 1 {
		gap = 1
	}

	return leftParts + strings.Repeat(" ", gap) + rightParts
}

// RenderOuterFrame wraps content in a btop-style outer frame with header in top border.
func RenderOuterFrame(header, body string, width, height int) string {
	bc := ColorBorder

	// Top border with embedded header
	topLeft := lipgloss.NewStyle().Foreground(bc).Render("╭─┤ ")
	topRight := lipgloss.NewStyle().Foreground(bc).Render(" ├")
	headerRendered := topLeft + header + topRight

	headerWidth := lipgloss.Width(headerRendered)
	topFill := width - headerWidth - 1
	if topFill < 0 {
		topFill = 0
	}
	topLine := headerRendered +
		lipgloss.NewStyle().Foreground(bc).Render(strings.Repeat("─", topFill)) +
		lipgloss.NewStyle().Foreground(bc).Render("╮")

	// Bottom border
	bottomLine := lipgloss.NewStyle().Foreground(bc).Render("╰" + strings.Repeat("─", width-2) + "╯")

	// Side borders for body
	bodyLines := strings.Split(body, "\n")
	var framedLines []string
	framedLines = append(framedLines, topLine)

	leftBar := lipgloss.NewStyle().Foreground(bc).Render("│")
	rightBar := lipgloss.NewStyle().Foreground(bc).Render("│")

	for _, line := range bodyLines {
		lineWidth := lipgloss.Width(line)
		padding := width - lineWidth - 2 // 2 for side borders
		if padding < 0 {
			padding = 0
		}
		framedLines = append(framedLines, leftBar+line+strings.Repeat(" ", padding)+rightBar)
	}

	framedLines = append(framedLines, bottomLine)
	return strings.Join(framedLines, "\n")
}

// RenderDivider renders a horizontal divider with optional embedded title.
func RenderDivider(title string, width int) string {
	bc := ColorBorder
	if title == "" {
		return lipgloss.NewStyle().Foreground(bc).Render("├" + strings.Repeat("─", width-2) + "┤")
	}

	titleRendered := lipgloss.NewStyle().Bold(true).Foreground(ColorFg).Render(title)
	left := lipgloss.NewStyle().Foreground(bc).Render("├─┤ ")
	right := lipgloss.NewStyle().Foreground(bc).Render(" ├")
	prefix := left + titleRendered + right

	prefixWidth := lipgloss.Width(prefix)
	remaining := width - prefixWidth - 1
	if remaining < 0 {
		remaining = 0
	}

	return prefix +
		lipgloss.NewStyle().Foreground(bc).Render(strings.Repeat("─", remaining)) +
		lipgloss.NewStyle().Foreground(bc).Render("┤")
}

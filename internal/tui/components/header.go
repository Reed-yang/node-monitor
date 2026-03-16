package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderHeader renders the top status bar.
func RenderHeader(nodes []model.NodeStatus, interval float64, width int) string {
	now := time.Now().Format("2006-01-02 15:04:05")

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

	utilColor := UtilColor(avgUtil)
	avgMemPct := 0.0
	if totalMem > 0 {
		avgMemPct = float64(totalMemUsed) / float64(totalMem) * 100
	}
	memColor := MemColor(avgMemPct)

	title := lipgloss.NewStyle().Bold(true).Render("🖥️  GPU Cluster Monitor")
	sep := lipgloss.NewStyle().Faint(true).Render("  │  ")

	nodeStr := fmt.Sprintf("📊 %d", onlineNodes)
	if offlineNodes > 0 {
		nodeStr += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("/%d", offlineNodes))
	}
	nodeStr += " nodes"

	utilStr := lipgloss.NewStyle().Bold(true).Foreground(utilColor).Render(fmt.Sprintf("⚡ %.0f%%", avgUtil))
	memStr := lipgloss.NewStyle().Bold(true).Foreground(memColor).Render(
		fmt.Sprintf("💾 %s/%s", model.FormatMemory(totalMemUsed), model.FormatMemory(totalMem)),
	)

	return title + sep +
		"🕐 " + now + sep +
		fmt.Sprintf("🔄 %.1fs", interval) + sep +
		nodeStr + sep +
		fmt.Sprintf("🎮 %d GPUs", totalGPUs) + sep +
		utilStr + sep +
		memStr
}

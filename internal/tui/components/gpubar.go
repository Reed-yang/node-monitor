package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderBar creates a progress bar string of given width.
func RenderBar(percent float64, width int, color lipgloss.Color) string {
	filled := int((percent / 100) * float64(width))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

// RenderGPURow renders a single GPU's info as a styled line for panel view.
func RenderGPURow(gpu model.GPUInfo, barWidth int) string {
	utilColor := UtilColor(float64(gpu.Utilization))
	memColor := MemColor(gpu.MemoryPercent())

	utilBar := RenderBar(float64(gpu.Utilization), barWidth, utilColor)
	memBar := RenderBar(gpu.MemoryPercent(), barWidth, memColor)

	utilPct := lipgloss.NewStyle().Bold(true).Foreground(utilColor).Render(fmt.Sprintf("%3d%%", gpu.Utilization))
	memStr := lipgloss.NewStyle().Bold(true).Foreground(memColor).Render(
		fmt.Sprintf("%s/%s", model.FormatMemory(gpu.MemoryUsed), model.FormatMemory(gpu.MemoryTotal)),
	)

	label := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("GPU%d ", gpu.Index))
	sep := lipgloss.NewStyle().Faint(true).Render(" │ ")

	return label + utilBar + " " + utilPct + sep + memBar + " " + memStr
}

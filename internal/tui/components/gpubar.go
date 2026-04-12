package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderGradientBar renders a btop-style gradient progress bar.
// Each filled character is independently colored along the gradient.
// Unfilled characters use a dark meter_bg color.
func RenderGradientBar(percent float64, width int, grad Gradient) string {
	if width <= 0 {
		return ""
	}
	filled := int((percent / 100) * float64(width))
	if filled > width {
		filled = width
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			// Each filled char gets its own color based on position
			threshold := i * 100 / width
			color := grad.ColorAt(threshold)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(ColorMeterBg).Render("░"))
		}
	}
	return b.String()
}

// RenderGPUHeatmap renders one colored block per GPU (mini heatmap strip).
func RenderGPUHeatmap(gpus []model.GPUInfo) string {
	var b strings.Builder
	for _, gpu := range gpus {
		color := UtilGradient.ColorAt(gpu.Utilization)
		b.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
	}
	return b.String()
}

// RenderGPURow renders a single GPU's info as a styled line for detail view.
// Format: GPU0 ■■■■■■■■■■░░░░░░░░░░ 87%  74.9G/79.6G
func RenderGPURow(gpu model.GPUInfo, barWidth int) string {
	utilBar := RenderGradientBar(float64(gpu.Utilization), barWidth, UtilGradient)
	utilColor := UtilColor(float64(gpu.Utilization))
	utilPct := lipgloss.NewStyle().Bold(true).Foreground(utilColor).Render(fmt.Sprintf("%3d%%", gpu.Utilization))

	memColor := MemColor(gpu.MemoryPercent())
	memStr := lipgloss.NewStyle().Foreground(memColor).Render(
		fmt.Sprintf("%s/%s", model.FormatMemory(gpu.MemoryUsed), model.FormatMemory(gpu.MemoryTotal)),
	)

	label := lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("%d", gpu.Index))
	sep := lipgloss.NewStyle().Foreground(ColorBorder).Render(" │ ")

	return " " + label + " " + utilBar + " " + utilPct + sep + memStr
}

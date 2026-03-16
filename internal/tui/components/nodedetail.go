package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderNodeDetail renders the detail drill-down view for a single node.
func RenderNodeDetail(node model.NodeStatus, sysInfo *model.SystemInfo, width int) string {
	if !node.IsOnline() {
		errMsg := "Offline"
		if node.Error != nil {
			errMsg = *node.Error
		}
		content := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("⚠ " + errMsg)
		return renderDetailPanel(node.Hostname, content, lipgloss.Color("#FF0000"), width)
	}

	var sections []string

	// GPU bars
	for _, gpu := range node.GPUs {
		sections = append(sections, RenderGPURow(gpu, 20))
	}

	// Processes
	allProcs := node.AllProcesses()
	if len(allProcs) > 0 {
		sections = append(sections, "")
		sections = append(sections, sectionHeader("Processes", width-6))
		hdr := fmt.Sprintf("  %-8s  %3s  %8s  %s", "USER", "GPU", "MEM", "CMD")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Faint(true).Render(hdr))

		for _, p := range allProcs {
			line := fmt.Sprintf("  %-8s  %3d  %7s  %s",
				p.User, p.GPUIndex, model.FormatMemory(p.MemoryMiB), p.Command)
			sections = append(sections, line)
		}
	}

	// System info
	if sysInfo != nil {
		sections = append(sections, "")
		sections = append(sections, sectionHeader("System", width-6))

		loadStr := fmt.Sprintf("  Load: %.2f / %.2f / %.2f", sysInfo.LoadAvg1, sysInfo.LoadAvg5, sysInfo.LoadAvg15)
		sections = append(sections, loadStr)

		if sysInfo.MemTotalBytes > 0 {
			memUsedG := float64(sysInfo.MemUsedBytes) / (1024 * 1024 * 1024)
			memTotalG := float64(sysInfo.MemTotalBytes) / (1024 * 1024 * 1024)
			sections = append(sections, fmt.Sprintf("  Memory: %.1fG / %.1fG", memUsedG, memTotalG))
		}

		if sysInfo.DriverVersion != "" {
			sections = append(sections, fmt.Sprintf("  Driver: %s", sysInfo.DriverVersion))
		}
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Faint(true).Italic(true).Render("  Esc to go back"))

	content := strings.Join(sections, "\n")
	borderColor := NodeBorderColor(node.AvgUtilization())
	return renderDetailPanel(node.Hostname, content, borderColor, width)
}

func renderDetailPanel(hostname, content string, borderColor lipgloss.Color, width int) string {
	panelWidth := width - 4
	if panelWidth > 80 {
		panelWidth = 80
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(panelWidth)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(borderColor).
		Width(panelWidth - 6).
		Align(lipgloss.Center).
		Render(hostname)

	return style.Render(title + "\n\n" + content)
}

func sectionHeader(title string, width int) string {
	line := strings.Repeat("─", width)
	return lipgloss.NewStyle().Faint(true).Render("  ── " + title + " " + line[:width-len(title)-6])
}

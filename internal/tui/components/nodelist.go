package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderPanelView renders all nodes as bordered panels with auto-column layout.
func RenderPanelView(nodes []model.NodeStatus, selectedIdx int, width int) string {
	minPanelWidth := 65
	numCols := width / minPanelWidth
	if numCols < 1 {
		numCols = 1
	}
	panelWidth := width/numCols - 2
	if panelWidth < minPanelWidth {
		panelWidth = minPanelWidth
	}

	var panels []string
	for i, node := range nodes {
		panels = append(panels, renderNodePanel(node, panelWidth, i == selectedIdx))
	}

	// Arrange in rows
	var rows []string
	for i := 0; i < len(panels); i += numCols {
		end := i + numCols
		if end > len(panels) {
			end = len(panels)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, panels[i:end]...)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

// renderNodePanel renders a single node as a bordered panel.
func renderNodePanel(node model.NodeStatus, width int, selected bool) string {
	borderColor := lipgloss.Color("#555555")
	var contentLines []string
	var titleLine string

	if !node.IsOnline() {
		errMsg := "Offline"
		if node.Error != nil && len(*node.Error) > 0 {
			errMsg = *node.Error
			if len(errMsg) > 40 {
				errMsg = errMsg[:40]
			}
		}
		titleLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")).Render("✗ " + node.Hostname)
		contentLines = append(contentLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("⚠ "+errMsg))
		borderColor = lipgloss.Color("#FF0000")
	} else if len(node.GPUs) == 0 {
		titleLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00")).Render("? " + node.Hostname)
		contentLines = append(contentLines, lipgloss.NewStyle().Faint(true).Render("No GPUs detected"))
		borderColor = lipgloss.Color("#FFFF00")
	} else {
		avgUtil := node.AvgUtilization()
		icon := NodeStatusIcon(avgUtil)
		borderColor = NodeBorderColor(avgUtil)

		titleLine = lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(icon + " " + node.Hostname)

		for _, gpu := range node.GPUs {
			contentLines = append(contentLines, RenderGPURow(gpu, 15))
		}

		// Summary
		utilColor := UtilColor(avgUtil)
		memPct := 0.0
		if node.TotalMemory() > 0 {
			memPct = float64(node.TotalMemoryUsed()) / float64(node.TotalMemory()) * 100
		}
		memColor := MemColor(memPct)

		contentLines = append(contentLines,
			lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", width-6)),
			fmt.Sprintf("Σ %d GPUs", node.TotalGPUs())+
				lipgloss.NewStyle().Faint(true).Render(" │ ")+
				lipgloss.NewStyle().Bold(true).Foreground(utilColor).Render(fmt.Sprintf("Util: %.0f%%", avgUtil))+
				lipgloss.NewStyle().Faint(true).Render(" │ ")+
				lipgloss.NewStyle().Bold(true).Foreground(memColor).Render(
					fmt.Sprintf("Mem: %s/%s", model.FormatMemory(node.TotalMemoryUsed()), model.FormatMemory(node.TotalMemory()))),
		)
	}

	// Center the title
	paddedTitle := lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).Render(titleLine)
	allContent := paddedTitle + "\n" + strings.Join(contentLines, "\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	if selected {
		style = style.BorderForeground(lipgloss.Color("#FFFFFF"))
	}

	return style.Render(allContent)
}

// RenderCompactView renders nodes as a dense table.
func RenderCompactView(nodes []model.NodeStatus, selectedIdx int, width int, showProcesses bool) string {
	dualColumn := width >= 120
	barWidth := 6

	var lines []string

	// Header
	if dualColumn {
		hdr := fmt.Sprintf("  %-12s  %2s  %4s  %-*s  %-11s  │  %2s  %4s  %-*s  %-11s",
			"Node", "#", "Utl", barWidth, "", "Mem",
			"#", "Utl", barWidth, "", "Mem")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render(hdr))
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", width)))
	} else {
		hdr := fmt.Sprintf("  %-14s  %2s  %4s  %-*s  %-11s  %-*s  %4s",
			"Node", "#", "Util", barWidth, "", "Memory", barWidth, "", "%")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Render(hdr))
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", width)))
	}

	for nodeIdx, node := range nodes {
		selected := nodeIdx == selectedIdx

		if !node.IsOnline() {
			errMsg := "Offline"
			if node.Error != nil && len(*node.Error) > 0 {
				errMsg = *node.Error
				if len(errMsg) > 20 {
					errMsg = errMsg[:20]
				}
			}
			name := "✗ " + node.Hostname
			if selected {
				name = lipgloss.NewStyle().Reverse(true).Render(name)
			} else {
				name = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(name)
			}
			line := fmt.Sprintf("  %-14s  %s", name, lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#FF0000")).Render(errMsg))
			lines = append(lines, line)
			continue
		}

		if len(node.GPUs) == 0 {
			name := "? " + node.Hostname
			line := fmt.Sprintf("  %-14s  %s", name, lipgloss.NewStyle().Faint(true).Render("No GPUs"))
			lines = append(lines, line)
			continue
		}

		icon := NodeStatusIcon(node.AvgUtilization())

		if dualColumn {
			for i := 0; i < len(node.GPUs); i += 2 {
				gpu1 := node.GPUs[i]
				nodeName := ""
				if i == 0 {
					nodeName = icon + " " + node.Hostname
				}
				if selected && i == 0 {
					nodeName = lipgloss.NewStyle().Reverse(true).Render(nodeName)
				}

				utilColor1 := UtilColor(float64(gpu1.Utilization))
				bar1 := RenderBar(float64(gpu1.Utilization), barWidth, utilColor1)
				pct1 := lipgloss.NewStyle().Bold(true).Foreground(utilColor1).Render(fmt.Sprintf("%3d%%", gpu1.Utilization))
				mem1 := fmt.Sprintf("%s/%s", model.FormatMemory(gpu1.MemoryUsed), model.FormatMemory(gpu1.MemoryTotal))

				right := ""
				if i+1 < len(node.GPUs) {
					gpu2 := node.GPUs[i+1]
					utilColor2 := UtilColor(float64(gpu2.Utilization))
					bar2 := RenderBar(float64(gpu2.Utilization), barWidth, utilColor2)
					pct2 := lipgloss.NewStyle().Bold(true).Foreground(utilColor2).Render(fmt.Sprintf("%3d%%", gpu2.Utilization))
					mem2 := fmt.Sprintf("%s/%s", model.FormatMemory(gpu2.MemoryUsed), model.FormatMemory(gpu2.MemoryTotal))
					right = fmt.Sprintf("  %2d  %s  %s  %s", gpu2.Index, pct2, bar2, mem2)
				}

				sep := lipgloss.NewStyle().Faint(true).Render("│")
				line := fmt.Sprintf("  %-12s  %2d  %s  %s  %-11s  %s%s",
					nodeName, gpu1.Index, pct1, bar1, mem1, sep, right)
				lines = append(lines, line)
			}
		} else {
			for i, gpu := range node.GPUs {
				nodeName := ""
				if i == 0 {
					nodeName = icon + " " + node.Hostname
				}
				if selected && i == 0 {
					nodeName = lipgloss.NewStyle().Reverse(true).Render(nodeName)
				}

				utilColor := UtilColor(float64(gpu.Utilization))
				memColor := MemColor(gpu.MemoryPercent())
				utilBar := RenderBar(float64(gpu.Utilization), barWidth, utilColor)
				memBar := RenderBar(gpu.MemoryPercent(), barWidth, memColor)
				pct := lipgloss.NewStyle().Bold(true).Foreground(utilColor).Render(fmt.Sprintf("%3d%%", gpu.Utilization))
				mem := fmt.Sprintf("%s/%s", model.FormatMemory(gpu.MemoryUsed), model.FormatMemory(gpu.MemoryTotal))
				memPct := lipgloss.NewStyle().Bold(true).Foreground(memColor).Render(fmt.Sprintf("%.0f%%", gpu.MemoryPercent()))

				line := fmt.Sprintf("  %-14s  %2d  %s  %s  %-11s  %s  %s",
					nodeName, gpu.Index, pct, utilBar, mem, memBar, memPct)
				lines = append(lines, line)
			}
		}

		// Process info
		if showProcesses && len(node.AllProcesses()) > 0 {
			procLine := renderProcessSummary(node)
			lines = append(lines, procLine)
		}
	}

	return strings.Join(lines, "\n")
}

func renderProcessSummary(node model.NodeStatus) string {
	userProcs := make(map[string][]model.GPUProcess)
	for _, p := range node.AllProcesses() {
		userProcs[p.User] = append(userProcs[p.User], p)
	}

	var users []string
	for user := range userProcs {
		users = append(users, user)
	}
	sort.Strings(users)

	var parts []string
	for _, user := range users {
		procs := userProcs[user]
		gpuSet := make(map[int]bool)
		totalMem := 0
		for _, p := range procs {
			gpuSet[p.GPUIndex] = true
			totalMem += p.MemoryMiB
		}
		var gpuIDs []int
		for id := range gpuSet {
			gpuIDs = append(gpuIDs, id)
		}
		sort.Ints(gpuIDs)
		gpuStrs := make([]string, len(gpuIDs))
		for i, id := range gpuIDs {
			gpuStrs[i] = fmt.Sprintf("%d", id)
		}
		parts = append(parts, fmt.Sprintf("%s[GPU %s]:%s", user, strings.Join(gpuStrs, ","), model.FormatMemory(totalMem)))
	}

	return lipgloss.NewStyle().Faint(true).Render("  📋 "+node.Hostname+" processes: ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Render(strings.Join(parts, " │ "))
}

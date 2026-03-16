package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderNodeDetail renders node detail for the bottom panel.
func RenderNodeDetail(node model.NodeStatus, sysInfo *model.SystemInfo, width int) string {
	if !node.IsOnline() {
		errMsg := "Offline"
		if node.Error != nil {
			errMsg = *node.Error
		}
		return lipgloss.NewStyle().Foreground(ColorRed).Render("  ⚠ " + errMsg)
	}

	innerWidth := width - 4
	var sections []string

	// Per-GPU bars in 2-column layout
	barWidth := 18
	colWidth := innerWidth / 2
	if colWidth < 40 {
		colWidth = innerWidth
	}

	for i := 0; i < len(node.GPUs); i += 2 {
		left := RenderGPURow(node.GPUs[i], barWidth)
		if i+1 < len(node.GPUs) {
			right := RenderGPURow(node.GPUs[i+1], barWidth)
			leftPad := colWidth - lipgloss.Width(left)
			if leftPad < 1 {
				leftPad = 1
			}
			sections = append(sections, left+strings.Repeat(" ", leftPad)+right)
		} else {
			sections = append(sections, left)
		}
	}

	// Processes for this node
	procs := node.AllProcesses()
	if len(procs) > 0 {
		sections = append(sections, "")

		// Aggregate by user
		type userAgg struct {
			User string
			GPUs []int
			Mem  int
			Cmd  string
		}
		aggMap := make(map[string]*userAgg)
		var users []string
		for _, p := range procs {
			if a, ok := aggMap[p.User]; ok {
				a.GPUs = appendUnique(a.GPUs, p.GPUIndex)
				a.Mem += p.MemoryMiB
			} else {
				aggMap[p.User] = &userAgg{
					User: p.User,
					GPUs: []int{p.GPUIndex},
					Mem:  p.MemoryMiB,
					Cmd:  p.Command,
				}
				users = append(users, p.User)
			}
		}
		sort.Strings(users)

		hdr := fmt.Sprintf(" %-8s  %-6s  %7s  %s", "USER", "GPU", "MEM", "CMD")
		sections = append(sections, lipgloss.NewStyle().Foreground(ColorDim).Bold(true).Render(hdr))

		for _, user := range users {
			a := aggMap[user]
			sort.Ints(a.GPUs)
			gpuStr := formatGPURange(a.GPUs)
			cmd := a.Cmd
			if len(cmd) > 40 {
				cmd = cmd[:38] + ".."
			}

			line := " " +
				lipgloss.NewStyle().Foreground(ColorAccent).Render(fmt.Sprintf("%-8s", a.User)) + "  " +
				lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("%-6s", gpuStr)) + "  " +
				lipgloss.NewStyle().Bold(true).Foreground(ColorFg).Render(fmt.Sprintf("%7s", model.FormatMemory(a.Mem))) + "  " +
				lipgloss.NewStyle().Foreground(ColorFg).Render(cmd)
			sections = append(sections, line)
		}
	}

	// System info
	if sysInfo != nil {
		sections = append(sections, "")
		var sysLine strings.Builder
		sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" Load: "))
		sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorFg).Render(
			fmt.Sprintf("%.1f/%.1f/%.1f", sysInfo.LoadAvg1, sysInfo.LoadAvg5, sysInfo.LoadAvg15)))

		if sysInfo.MemTotalBytes > 0 {
			memG := float64(sysInfo.MemUsedBytes) / (1024 * 1024 * 1024)
			totalG := float64(sysInfo.MemTotalBytes) / (1024 * 1024 * 1024)
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render("  │  "))
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("RAM: "))
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorFg).Render(
				fmt.Sprintf("%.0fG/%.0fG", memG, totalG)))
		}

		if sysInfo.DriverVersion != "" {
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render("  │  "))
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("Driver: "))
			sysLine.WriteString(lipgloss.NewStyle().Foreground(ColorFg).Render(sysInfo.DriverVersion))
		}

		sections = append(sections, sysLine.String())
	}

	// Esc hint
	sections = append(sections, lipgloss.NewStyle().Foreground(ColorDim).Italic(true).Render("  Esc to go back"))

	return strings.Join(sections, "\n")
}

func appendUnique(slice []int, val int) []int {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

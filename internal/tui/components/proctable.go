package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderProcessTable renders the global process table.
func RenderProcessTable(nodes []model.NodeStatus, width int, maxRows int) string {
	// Collect all processes with node info
	type procEntry struct {
		User    string
		Node    string
		GPUs    string
		Mem     string
		Command string
	}

	// Aggregate by user+node
	type aggKey struct{ user, node string }
	agg := make(map[aggKey]*procEntry)
	var keys []aggKey

	for _, n := range nodes {
		if !n.IsOnline() {
			continue
		}
		userProcs := make(map[string][]model.GPUProcess)
		for _, g := range n.GPUs {
			for _, p := range g.Processes {
				userProcs[p.User] = append(userProcs[p.User], p)
			}
		}
		for user, procs := range userProcs {
			key := aggKey{user, n.Hostname}
			gpuSet := make(map[int]bool)
			totalMem := 0
			cmd := ""
			for _, p := range procs {
				gpuSet[p.GPUIndex] = true
				totalMem += p.MemoryMiB
				if cmd == "" {
					cmd = p.Command
				}
			}
			var gpuIDs []int
			for id := range gpuSet {
				gpuIDs = append(gpuIDs, id)
			}
			sort.Ints(gpuIDs)
			gpuStr := formatGPURange(gpuIDs)

			entry := &procEntry{
				User:    user,
				Node:    n.Hostname,
				GPUs:    gpuStr,
				Mem:     model.FormatMemory(totalMem),
				Command: cmd,
			}
			if _, ok := agg[key]; !ok {
				keys = append(keys, key)
			}
			agg[key] = entry
		}
	}

	// Sort by memory descending (parse back)
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].node < keys[j].node || (keys[i].node == keys[j].node && keys[i].user < keys[j].user)
	})

	if len(keys) == 0 {
		return lipgloss.NewStyle().Foreground(ColorDim).Render("  No GPU processes running")
	}

	innerWidth := width - 4

	// Header
	hdr := fmt.Sprintf(" %-8s  %-12s  %-6s  %7s  %s", "USER", "NODE", "GPU", "MEM", "CMD")
	hdrStyled := lipgloss.NewStyle().Bold(true).Foreground(ColorDim).Render(hdr)

	var lines []string
	lines = append(lines, hdrStyled)

	for i, key := range keys {
		if maxRows > 0 && i >= maxRows {
			lines = append(lines, lipgloss.NewStyle().Foreground(ColorDim).Render(
				fmt.Sprintf("  ... and %d more", len(keys)-maxRows)))
			break
		}
		e := agg[key]
		cmd := e.Command
		maxCmd := innerWidth - 40
		if maxCmd < 10 {
			maxCmd = 10
		}
		if len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-2] + ".."
		}

		userStyle := lipgloss.NewStyle().Foreground(ColorAccent)
		nodeStyle := lipgloss.NewStyle().Foreground(ColorDim)
		gpuStyle := lipgloss.NewStyle().Foreground(ColorGreen)
		memStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorFg)

		line := " " + userStyle.Render(fmt.Sprintf("%-8s", e.User)) + "  " +
			nodeStyle.Render(fmt.Sprintf("%-12s", e.Node)) + "  " +
			gpuStyle.Render(fmt.Sprintf("%-6s", e.GPUs)) + "  " +
			memStyle.Render(fmt.Sprintf("%7s", e.Mem)) + "  " +
			lipgloss.NewStyle().Foreground(ColorFg).Render(cmd)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// formatGPURange formats GPU indices compactly: [0,1,2,3] -> "0-3", [0,2,5] -> "0,2,5"
func formatGPURange(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return fmt.Sprintf("%d", ids[0])
	}

	// Check if contiguous
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

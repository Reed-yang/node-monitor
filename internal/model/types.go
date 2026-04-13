package model

import (
	"fmt"
	"strings"
)

// GPUProcess represents a running process on a GPU.
type GPUProcess struct {
	PID       int
	User      string
	GPUIndex  int
	MemoryMiB int
	Command   string
}

// GPUInfo represents a single GPU's status.
type GPUInfo struct {
	Index       int
	Utilization int // 0-100
	MemoryUsed  int // MiB
	MemoryTotal int // MiB
	Name        string // GPU model name (e.g., "NVIDIA A100-SXM4-80GB")
	Processes   []GPUProcess
}

// ShortName returns abbreviated GPU model name (e.g., "A100").
func (g GPUInfo) ShortName() string {
	name := g.Name
	// Common patterns: "NVIDIA A100-SXM4-80GB" -> "A100", "Tesla V100-SXM2-32GB" -> "V100"
	for _, prefix := range []string{"NVIDIA ", "Tesla "} {
		name = strings.TrimPrefix(name, prefix)
	}
	if idx := strings.IndexAny(name, "- "); idx > 0 {
		name = name[:idx]
	}
	if name == "" {
		return "GPU"
	}
	return name
}

// MemoryPercent returns memory usage as a percentage.
func (g GPUInfo) MemoryPercent() float64 {
	if g.MemoryTotal == 0 {
		return 0.0
	}
	return float64(g.MemoryUsed) / float64(g.MemoryTotal) * 100
}

// NodeStatus represents a node and all its GPUs.
type NodeStatus struct {
	Hostname string
	GPUs     []GPUInfo
	Error    *string // nil means online
}

func (n NodeStatus) IsOnline() bool        { return n.Error == nil }
func (n NodeStatus) TotalGPUs() int        { return len(n.GPUs) }
func (n NodeStatus) TotalMemoryUsed() int  { return sumField(n.GPUs, func(g GPUInfo) int { return g.MemoryUsed }) }
func (n NodeStatus) TotalMemory() int      { return sumField(n.GPUs, func(g GPUInfo) int { return g.MemoryTotal }) }

func (n NodeStatus) AvgUtilization() float64 {
	if len(n.GPUs) == 0 {
		return 0.0
	}
	total := 0
	for _, g := range n.GPUs {
		total += g.Utilization
	}
	return float64(total) / float64(len(n.GPUs))
}

// GPUModelSummary returns e.g. "8×A100"
func (n NodeStatus) GPUModelSummary() string {
	if len(n.GPUs) == 0 {
		return ""
	}
	name := n.GPUs[0].ShortName()
	return fmt.Sprintf("%d×%s", len(n.GPUs), name)
}

// ActiveUsers returns unique usernames with GPU processes.
func (n NodeStatus) ActiveUsers() []string {
	seen := make(map[string]bool)
	var users []string
	for _, g := range n.GPUs {
		for _, p := range g.Processes {
			if !seen[p.User] {
				seen[p.User] = true
				users = append(users, p.User)
			}
		}
	}
	return users
}

func (n NodeStatus) AllProcesses() []GPUProcess {
	var procs []GPUProcess
	for _, g := range n.GPUs {
		procs = append(procs, g.Processes...)
	}
	return procs
}

// SystemInfo holds optional system-level details for the detail view.
type SystemInfo struct {
	LoadAvg1      float64
	LoadAvg5      float64
	LoadAvg15     float64
	MemTotalBytes int64
	MemUsedBytes  int64
	DriverVersion string
}

// FormatMemory formats MiB to human-readable string.
func FormatMemory(mib int) string {
	if mib >= 1024 {
		return fmt.Sprintf("%.1fG", float64(mib)/1024)
	}
	return fmt.Sprintf("%dM", mib)
}

func sumField(gpus []GPUInfo, f func(GPUInfo) int) int {
	total := 0
	for _, g := range gpus {
		total += f(g)
	}
	return total
}

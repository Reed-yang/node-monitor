# Node Monitor Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the Python node-monitor CLI in Go to produce a single static binary with an interactive TUI, SSH connection pooling, and config file support.

**Architecture:** Bubble Tea (Elm architecture) drives the TUI loop. SSH queries run as goroutines feeding results back via Bubble Tea messages. Cobra handles CLI parsing, Viper merges config file + CLI flags. All business logic lives in `internal/` packages with clear boundaries.

**Tech Stack:** Go 1.22+, Bubble Tea, Lip Gloss, Bubbles, Cobra, Viper, golang.org/x/crypto/ssh, kevinburke/ssh_config

**Spec:** `docs/superpowers/specs/2026-03-16-go-rewrite-design.md`

---

## Chunk 1: Foundation (Scaffolding, Model, Config)

### Task 1: Project Scaffolding

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `internal/model/types.go`

- [ ] **Step 1: Initialize Go module and directory structure**

```bash
cd /mnt/novita2/siyuan/workspace/node-monitor
git checkout -b feat/go-rewrite
mkdir -p cmd internal/{config,ssh,slurm,model,tui/components} configs
go mod init github.com/siyuan/node-monitor
```

- [ ] **Step 2: Install all dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get golang.org/x/crypto/ssh@latest
go get github.com/kevinburke/ssh_config@latest
go get golang.org/x/term@latest
```

- [ ] **Step 3: Create main.go with version embedding**

```go
// main.go
package main

import "github.com/siyuan/node-monitor/cmd"

// Set via ldflags: -X main.version=v1.0.0
var version = "dev"

func main() {
	cmd.Execute(version)
}
```

- [ ] **Step 4: Create minimal cmd/root.go to verify build**

```go
// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var appVersion string

var rootCmd = &cobra.Command{
	Use:   "node-monitor",
	Short: "GPU Cluster Monitor - Monitor GPU resources across Slurm nodes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("node-monitor", appVersion)
	},
}

func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verify build succeeds**

Run: `go build -o node-monitor .`
Expected: Binary `node-monitor` created, runs and prints version.

- [ ] **Step 6: Commit**

```bash
git add main.go cmd/root.go go.mod go.sum
git commit -m "feat: scaffold Go project with Cobra CLI"
```

---

### Task 2: Data Model

**Files:**
- Create: `internal/model/types.go`
- Create: `internal/model/types_test.go`

- [ ] **Step 1: Write tests for model types**

```go
// internal/model/types_test.go
package model

import "testing"

func TestGPUInfo_MemoryPercent(t *testing.T) {
	tests := []struct {
		name     string
		gpu      GPUInfo
		expected float64
	}{
		{"normal usage", GPUInfo{Index: 0, Utilization: 87, MemoryUsed: 74900, MemoryTotal: 79600}, 94.09},
		{"zero total", GPUInfo{Index: 0, Utilization: 0, MemoryUsed: 0, MemoryTotal: 0}, 0.0},
		{"empty", GPUInfo{Index: 0, Utilization: 0, MemoryUsed: 0, MemoryTotal: 1024}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gpu.MemoryPercent()
			// Allow 0.1% tolerance
			if got < tt.expected-0.1 || got > tt.expected+0.1 {
				t.Errorf("MemoryPercent() = %v, want ~%v", got, tt.expected)
			}
		})
	}
}

func TestNodeStatus_Aggregates(t *testing.T) {
	node := NodeStatus{
		Hostname: "visko-1",
		GPUs: []GPUInfo{
			{Index: 0, Utilization: 80, MemoryUsed: 40000, MemoryTotal: 80000},
			{Index: 1, Utilization: 60, MemoryUsed: 20000, MemoryTotal: 80000},
		},
	}

	if node.TotalGPUs() != 2 {
		t.Errorf("TotalGPUs() = %d, want 2", node.TotalGPUs())
	}
	if node.AvgUtilization() != 70.0 {
		t.Errorf("AvgUtilization() = %f, want 70.0", node.AvgUtilization())
	}
	if node.TotalMemoryUsed() != 60000 {
		t.Errorf("TotalMemoryUsed() = %d, want 60000", node.TotalMemoryUsed())
	}
	if node.TotalMemory() != 160000 {
		t.Errorf("TotalMemory() = %d, want 160000", node.TotalMemory())
	}
	if !node.IsOnline() {
		t.Error("IsOnline() = false, want true")
	}
}

func TestNodeStatus_Offline(t *testing.T) {
	errMsg := "Connection timed out"
	node := NodeStatus{
		Hostname: "visko-2",
		GPUs:     nil,
		Error:    &errMsg,
	}
	if node.IsOnline() {
		t.Error("IsOnline() = true, want false")
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		mib      int
		expected string
	}{
		{79600, "77.7G"},
		{512, "512M"},
		{1024, "1.0G"},
		{0, "0M"},
	}
	for _, tt := range tests {
		got := FormatMemory(tt.mib)
		if got != tt.expected {
			t.Errorf("FormatMemory(%d) = %q, want %q", tt.mib, got, tt.expected)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/model/ -v`
Expected: FAIL — types not defined yet.

- [ ] **Step 3: Implement model types**

```go
// internal/model/types.go
package model

import "fmt"

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
	Processes   []GPUProcess
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/ -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/
git commit -m "feat: add data model types with tests"
```

---

### Task 3: Configuration System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `configs/default.toml`

- [ ] **Step 1: Write tests for config loading**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Interval != 2.0 {
		t.Errorf("Interval = %f, want 2.0", cfg.Interval)
	}
	if cfg.Workers != 8 {
		t.Errorf("Workers = %d, want 8", cfg.Workers)
	}
	if cfg.View != "panel" {
		t.Errorf("View = %q, want panel", cfg.View)
	}
	if cfg.SSH.ConnectTimeout != 5 {
		t.Errorf("SSH.ConnectTimeout = %d, want 5", cfg.SSH.ConnectTimeout)
	}
	if cfg.SSH.CommandTimeout != 10 {
		t.Errorf("SSH.CommandTimeout = %d, want 10", cfg.SSH.CommandTimeout)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	content := `
nodes = ["node-1", "node-2"]
interval = 5.0
workers = 4
view = "compact"
processes = true

[ssh]
connect_timeout = 10
command_timeout = 20
user = "testuser"

[groups]
train = ["node-1"]
inference = ["node-2"]
`
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	if len(cfg.Nodes) != 2 || cfg.Nodes[0] != "node-1" {
		t.Errorf("Nodes = %v, want [node-1, node-2]", cfg.Nodes)
	}
	if cfg.Interval != 5.0 {
		t.Errorf("Interval = %f, want 5.0", cfg.Interval)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, want 4", cfg.Workers)
	}
	if cfg.View != "compact" {
		t.Errorf("View = %q, want compact", cfg.View)
	}
	if !cfg.Processes {
		t.Error("Processes = false, want true")
	}
	if cfg.SSH.ConnectTimeout != 10 {
		t.Errorf("SSH.ConnectTimeout = %d, want 10", cfg.SSH.ConnectTimeout)
	}
	if cfg.SSH.User != "testuser" {
		t.Errorf("SSH.User = %q, want testuser", cfg.SSH.User)
	}
	if trains, ok := cfg.Groups["train"]; !ok || len(trains) != 1 {
		t.Errorf("Groups[train] = %v, want [node-1]", trains)
	}
}

func TestResolveNodes_GroupOverride(t *testing.T) {
	cfg := Defaults()
	cfg.Nodes = []string{"a", "b"}
	cfg.Groups = map[string][]string{
		"train": {"x", "y"},
	}

	nodes := cfg.ResolveNodes("train")
	if len(nodes) != 2 || nodes[0] != "x" {
		t.Errorf("ResolveNodes(train) = %v, want [x, y]", nodes)
	}

	nodes = cfg.ResolveNodes("")
	if len(nodes) != 2 || nodes[0] != "a" {
		t.Errorf("ResolveNodes('') = %v, want [a, b]", nodes)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL — config package not implemented.

- [ ] **Step 3: Implement config package**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type SSHConfig struct {
	ConnectTimeout int    `mapstructure:"connect_timeout"`
	CommandTimeout int    `mapstructure:"command_timeout"`
	IdentityFile   string `mapstructure:"identity_file"`
	User           string `mapstructure:"user"`
}

type Config struct {
	Nodes     []string            `mapstructure:"nodes"`
	Interval  float64             `mapstructure:"interval"`
	Workers   int                 `mapstructure:"workers"`
	View      string              `mapstructure:"view"`
	Processes bool                `mapstructure:"processes"`
	Debug     bool                `mapstructure:"debug"`
	Static    bool                `mapstructure:"static"`
	Compact   bool                `mapstructure:"compact"`
	Group     string              `mapstructure:"group"`
	SSH       SSHConfig           `mapstructure:"ssh"`
	Groups    map[string][]string `mapstructure:"groups"`
}

func Defaults() Config {
	return Config{
		Interval: 2.0,
		Workers:  8,
		View:     "panel",
		SSH: SSHConfig{
			ConnectTimeout: 5,
			CommandTimeout: 10,
		},
		Groups: make(map[string][]string),
	}
}

func LoadFromFile(path string) (Config, error) {
	cfg := Defaults()

	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return cfg, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}

	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}

	return cfg, nil
}

// Load tries to load from the default config path. Returns defaults if no file found.
func Load() Config {
	cfg := Defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".config", "node-monitor", "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		return cfg
	}
	return loaded
}

// ResolveNodes returns the node list based on group selection.
// If group is specified and exists, use that group's nodes.
// Otherwise, return the configured nodes list (may be empty, meaning auto-detect from Slurm).
func (c Config) ResolveNodes(group string) []string {
	if group != "" {
		if nodes, ok := c.Groups[group]; ok {
			return nodes
		}
	}
	return c.Nodes
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: All PASS.

- [ ] **Step 5: Create example config file**

```toml
# configs/default.toml
# Node Monitor configuration
# Copy to ~/.config/node-monitor/config.toml

# Default node list (omit to auto-detect from Slurm)
# nodes = ["visko-1", "visko-2", "visko-3"]

# Refresh interval (seconds)
interval = 2.0

# Max parallel SSH connections
workers = 8

# Default view mode: "panel" | "compact"
view = "panel"

# Show process info
processes = false

# SSH configuration
[ssh]
connect_timeout = 5   # SSH connection timeout (seconds)
command_timeout = 10  # Command execution timeout (seconds)
# identity_file = ""  # Private key path (empty = default)
# user = ""           # SSH username (empty = current user)

# Node groups (optional)
# [groups]
# train = ["visko-1", "visko-2", "visko-3", "visko-4"]
# inference = ["infer-1", "infer-2"]
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/ configs/
git commit -m "feat: add config system with Viper (TOML support)"
```

---

## Chunk 2: Data Layer (Slurm Detection, SSH Queries, Parsing)

### Task 4: Slurm Node Detection

**Files:**
- Create: `internal/slurm/detect.go`
- Create: `internal/slurm/detect_test.go`

- [ ] **Step 1: Write tests for parsing logic**

The actual `sinfo` call requires a Slurm cluster, so we test the parsing function separately.

```go
// internal/slurm/detect_test.go
package slurm

import "testing"

func TestParseSinfoOutput(t *testing.T) {
	output := "visko-1\nvisko-2\nvisko-1\nvisko-3\n\n"
	nodes := parseSinfoOutput(output)

	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	// Should be sorted and deduplicated
	expected := []string{"visko-1", "visko-2", "visko-3"}
	for i, n := range nodes {
		if n != expected[i] {
			t.Errorf("node[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestParseSinfoOutput_Empty(t *testing.T) {
	nodes := parseSinfoOutput("")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

func TestParseNodeList(t *testing.T) {
	nodes := ParseNodeList("visko-1, visko-2 , visko-3")
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	if nodes[0] != "visko-1" || nodes[2] != "visko-3" {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

func TestParseNodeList_Empty(t *testing.T) {
	nodes := ParseNodeList("")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/slurm/ -v`
Expected: FAIL.

- [ ] **Step 3: Implement slurm detection**

```go
// internal/slurm/detect.go
package slurm

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// DetectNodes runs sinfo to discover Slurm nodes.
func DetectNodes() ([]string, error) {
	ctx_timeout := 10 * time.Second
	cmd := exec.Command("sinfo", "-h", "-o", "%n")

	// Use a simple timeout approach
	done := make(chan error, 1)
	var output []byte
	var cmdErr error

	go func() {
		output, cmdErr = cmd.Output()
		done <- cmdErr
	}()

	select {
	case <-time.After(ctx_timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("sinfo command timed out")
	case err := <-done:
		if err != nil {
			if execErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("sinfo failed: %s", string(execErr.Stderr))
			}
			if _, ok := err.(*exec.Error); ok {
				return nil, fmt.Errorf("slurm is not installed or sinfo is not in PATH")
			}
			return nil, fmt.Errorf("sinfo failed: %w", err)
		}
	}

	nodes := parseSinfoOutput(string(output))
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found from sinfo")
	}
	return nodes, nil
}

func parseSinfoOutput(output string) []string {
	seen := make(map[string]bool)
	var nodes []string
	for _, line := range strings.Split(output, "\n") {
		node := strings.TrimSpace(line)
		if node != "" && !seen[node] {
			seen[node] = true
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	return nodes
}

// ParseNodeList parses a comma-separated node list string.
func ParseNodeList(s string) []string {
	var nodes []string
	for _, n := range strings.Split(s, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			nodes = append(nodes, n)
		}
	}
	return nodes
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/slurm/ -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/slurm/
git commit -m "feat: add Slurm node detection and node list parsing"
```

---

### Task 5: SSH Connection Pool and GPU Querying

**Files:**
- Create: `internal/ssh/pool.go`
- Create: `internal/ssh/query.go`
- Create: `internal/ssh/parse.go`
- Create: `internal/ssh/parse_test.go`

- [ ] **Step 1: Write tests for nvidia-smi output parsing**

This is the most testable part — parsing CSV output into model types.

```go
// internal/ssh/parse_test.go
package ssh

import (
	"testing"
)

func TestParseGPUOutput(t *testing.T) {
	output := "0, 87, 74900, 79600\n1, 78, 75000, 79600\n2, 100, 76500, 79600\n"
	gpus := parseGPUOutput(output)

	if len(gpus) != 3 {
		t.Fatalf("got %d GPUs, want 3", len(gpus))
	}

	if gpus[0].Index != 0 || gpus[0].Utilization != 87 || gpus[0].MemoryUsed != 74900 {
		t.Errorf("GPU0 = %+v", gpus[0])
	}
	if gpus[2].Utilization != 100 {
		t.Errorf("GPU2 utilization = %d, want 100", gpus[2].Utilization)
	}
}

func TestParseGPUOutput_Empty(t *testing.T) {
	gpus := parseGPUOutput("")
	if len(gpus) != 0 {
		t.Errorf("got %d GPUs, want 0", len(gpus))
	}
}

func TestParseGPUOutput_MalformedLine(t *testing.T) {
	output := "0, 87, 74900, 79600\nbadline\n1, 50, 1000, 2000\n"
	gpus := parseGPUOutput(output)
	if len(gpus) != 2 {
		t.Errorf("got %d GPUs, want 2 (skipping malformed)", len(gpus))
	}
}

func TestParseDetailOutput(t *testing.T) {
	output := `0, 87, 74900, 79600
1, 78, 75000, 79600
---PROCESSES---
GPU-uuid-aaa, 1234, 5000, python
GPU-uuid-bbb, 5678, 3000, train.py
---GPU_UUID_MAP---
0, GPU-uuid-aaa
1, GPU-uuid-bbb
---USERS---
1234 alice
5678 bob
---SYSTEM---
1.23 2.34 3.45 2/100 12345
              total        used        free      shared  buff/cache   available
Mem:    540000000000 270000000000 135000000000  1000000 134000000000 268000000000
NVIDIA UNIX x86_64 Kernel Module  535.129.03  Tue Oct 17 11:42:00 UTC 2023`

	result := parseDetailOutput(output)

	if len(result.GPUs) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(result.GPUs))
	}

	// Check processes were assigned to GPUs
	gpu0Procs := result.GPUs[0].Processes
	if len(gpu0Procs) != 1 || gpu0Procs[0].User != "alice" {
		t.Errorf("GPU0 processes = %+v, want 1 process by alice", gpu0Procs)
	}

	gpu1Procs := result.GPUs[1].Processes
	if len(gpu1Procs) != 1 || gpu1Procs[0].User != "bob" {
		t.Errorf("GPU1 processes = %+v, want 1 process by bob", gpu1Procs)
	}

	// Check system info
	if result.System == nil {
		t.Fatal("System info is nil")
	}
	if result.System.LoadAvg1 < 1.22 || result.System.LoadAvg1 > 1.24 {
		t.Errorf("LoadAvg1 = %f, want ~1.23", result.System.LoadAvg1)
	}
	if result.System.DriverVersion != "535.129.03" {
		t.Errorf("DriverVersion = %q, want 535.129.03", result.System.DriverVersion)
	}
}

func TestParseDetailOutput_NoProcesses(t *testing.T) {
	output := `0, 87, 74900, 79600
---PROCESSES---

---GPU_UUID_MAP---
0, GPU-uuid-aaa
---USERS---
1234 root
---SYSTEM---
0.50 0.60 0.70 1/50 999
              total        used        free      shared  buff/cache   available
Mem:    540000000000 100000000000 400000000000  1000000  40000000000 430000000000
N/A`

	result := parseDetailOutput(output)
	if len(result.GPUs) != 1 {
		t.Fatalf("got %d GPUs, want 1", len(result.GPUs))
	}
	if len(result.GPUs[0].Processes) != 0 {
		t.Errorf("expected 0 processes, got %d", len(result.GPUs[0].Processes))
	}
	if result.System.DriverVersion != "N/A" {
		t.Errorf("DriverVersion = %q, want N/A", result.System.DriverVersion)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ssh/ -v`
Expected: FAIL.

- [ ] **Step 3: Implement parse.go**

```go
// internal/ssh/parse.go
package ssh

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/siyuan/node-monitor/internal/model"
)

// DetailResult holds the parsed output from a detail query.
type DetailResult struct {
	GPUs   []model.GPUInfo
	System *model.SystemInfo
}

func parseGPUOutput(output string) []model.GPUInfo {
	var gpus []model.GPUInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		idx, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		util, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		memUsed, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
		memTotal, err4 := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		gpus = append(gpus, model.GPUInfo{
			Index:       idx,
			Utilization: util,
			MemoryUsed:  memUsed,
			MemoryTotal: memTotal,
		})
	}
	return gpus
}

func parseDetailOutput(output string) DetailResult {
	result := DetailResult{}

	sections := splitSections(output)

	result.GPUs = parseGPUOutput(sections["gpu"])

	// Build UUID-to-index map
	uuidToIdx := make(map[string]int)
	for _, line := range strings.Split(sections["uuid_map"], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		uuidToIdx[strings.TrimSpace(parts[1])] = idx
	}

	// Build PID-to-user map
	pidToUser := make(map[int]string)
	for _, line := range strings.Split(sections["users"], "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pidToUser[pid] = fields[1]
	}

	// Parse processes and assign to GPUs
	gpuMap := make(map[int]*model.GPUInfo)
	for i := range result.GPUs {
		gpuMap[result.GPUs[i].Index] = &result.GPUs[i]
	}

	for _, line := range strings.Split(sections["processes"], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		gpuUUID := strings.TrimSpace(parts[0])
		pid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		mem, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		cmdRaw := strings.TrimSpace(parts[3])
		// Get basename and truncate
		cmd := cmdRaw
		if idx := strings.LastIndex(cmd, "/"); idx >= 0 {
			cmd = cmd[idx+1:]
		}
		if len(cmd) > 20 {
			cmd = cmd[:20]
		}

		gpuIdx, ok := uuidToIdx[gpuUUID]
		if !ok {
			continue
		}
		user := pidToUser[pid]
		if user == "" {
			user = "?"
		}
		if len(user) > 8 {
			user = user[:8]
		}

		if gpu, ok := gpuMap[gpuIdx]; ok {
			gpu.Processes = append(gpu.Processes, model.GPUProcess{
				PID:       pid,
				User:      user,
				GPUIndex:  gpuIdx,
				MemoryMiB: mem,
				Command:   cmd,
			})
		}
	}

	// Parse system info
	result.System = parseSystemInfo(sections["system"])

	return result
}

func splitSections(output string) map[string]string {
	sections := map[string]string{
		"gpu":      "",
		"processes": "",
		"uuid_map": "",
		"users":    "",
		"system":   "",
	}

	markers := []struct {
		marker string
		key    string
	}{
		{"---PROCESSES---", "processes"},
		{"---GPU_UUID_MAP---", "uuid_map"},
		{"---USERS---", "users"},
		{"---SYSTEM---", "system"},
	}

	remaining := output

	// Extract GPU section (everything before first marker)
	if idx := strings.Index(remaining, "---PROCESSES---"); idx >= 0 {
		sections["gpu"] = remaining[:idx]
		remaining = remaining[idx:]
	} else {
		sections["gpu"] = remaining
		return sections
	}

	// Extract each subsequent section
	for i, m := range markers {
		start := strings.Index(remaining, m.marker)
		if start < 0 {
			continue
		}
		contentStart := start + len(m.marker)

		// Find end: next marker or end of string
		end := len(remaining)
		if i+1 < len(markers) {
			if nextIdx := strings.Index(remaining[contentStart:], markers[i+1].marker); nextIdx >= 0 {
				end = contentStart + nextIdx
			}
		}

		sections[m.key] = strings.TrimSpace(remaining[contentStart:end])
		remaining = remaining[end:]
	}

	return sections
}

func parseSystemInfo(section string) *model.SystemInfo {
	if section == "" {
		return nil
	}

	info := &model.SystemInfo{}
	lines := strings.Split(section, "\n")

	// Line 0: /proc/loadavg -> "1.23 2.34 3.45 2/100 12345"
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		if len(fields) >= 3 {
			info.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
			info.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
			info.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	// Line 1-2: free -b output (header + Mem: line)
	// Find line starting with "Mem:"
	for _, line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				info.MemTotalBytes, _ = strconv.ParseInt(fields[1], 10, 64)
				info.MemUsedBytes, _ = strconv.ParseInt(fields[2], 10, 64)
			}
			break
		}
	}

	// Last non-empty line: driver version
	// Pattern: "NVIDIA UNIX x86_64 Kernel Module  535.129.03  ..." or "N/A"
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if line == "N/A" {
			info.DriverVersion = "N/A"
		} else {
			// Extract version number from nvidia driver line
			re := regexp.MustCompile(`\d+\.\d+\.\d+`)
			if m := re.FindString(line); m != "" {
				info.DriverVersion = m
			} else {
				info.DriverVersion = line
			}
		}
		break
	}

	return info
}

// ListViewCommand returns the nvidia-smi command for list view queries.
func ListViewCommand() string {
	return "nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits"
}

// DetailViewCommand returns the batched command for detail view queries.
func DetailViewCommand() string {
	return fmt.Sprintf(`%s && echo '---PROCESSES---' && nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory,process_name --format=csv,noheader,nounits 2>/dev/null || true && echo '---GPU_UUID_MAP---' && nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits 2>/dev/null || true && echo '---USERS---' && ps -eo pid,user --no-headers 2>/dev/null || true && echo '---SYSTEM---' && cat /proc/loadavg && free -b | head -2 && cat /proc/driver/nvidia/version 2>/dev/null || echo 'N/A'`,
		ListViewCommand())
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ssh/ -v`
Expected: All PASS.

- [ ] **Step 5: Implement SSH connection pool (pool.go)**

```go
// internal/ssh/pool.go
package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	sshconfig "github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Pool manages persistent SSH connections to nodes.
type Pool struct {
	mu          sync.Mutex
	connections map[string]*ssh.Client
	timeout     time.Duration
	user        string
	identityFile string
}

// NewPool creates a new SSH connection pool.
func NewPool(connectTimeout int, user, identityFile string) *Pool {
	return &Pool{
		connections:  make(map[string]*ssh.Client),
		timeout:      time.Duration(connectTimeout) * time.Second,
		user:         user,
		identityFile: identityFile,
	}
}

// getClient returns a cached or new SSH client for the given host.
func (p *Pool) getClient(host string) (*ssh.Client, error) {
	p.mu.Lock()
	if client, ok := p.connections[host]; ok {
		p.mu.Unlock()
		// Test if connection is still alive
		_, _, err := client.SendRequest("keepalive@node-monitor", true, nil)
		if err == nil {
			return client, nil
		}
		// Connection is dead, remove it
		p.mu.Lock()
		delete(p.connections, host)
		client.Close()
		p.mu.Unlock()
	} else {
		p.mu.Unlock()
	}

	// Resolve SSH config for this host
	user := p.user
	if user == "" {
		user = sshconfig.Get(host, "User")
	}
	if user == "" {
		user = os.Getenv("USER")
	}

	port := sshconfig.Get(host, "Port")
	if port == "" {
		port = "22"
	}

	hostname := sshconfig.Get(host, "Hostname")
	if hostname == "" {
		hostname = host
	}

	// Build auth methods
	var authMethods []ssh.AuthMethod

	// Try SSH agent first (store conn to close later with the pool)
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if agentConn, err := net.Dial("unix", sock); err == nil {
			authMethods = append(authMethods, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
			// Note: agentConn stays open for the pool's lifetime — closed when pool.Close() is called.
			// For simplicity, agent connections are long-lived (one per pool, not per host).
		}
	}

	// Try identity file — check configured path, then common defaults
	keyPath := p.identityFile
	if keyPath == "" {
		keyPath = sshconfig.Get(host, "IdentityFile")
	}
	// Expand ~ in path
	if len(keyPath) > 0 && keyPath[0] == '~' {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[1:])
	}
	// Try configured key, then fall back to common defaults
	keyPaths := []string{}
	if keyPath != "" {
		keyPaths = append(keyPaths, keyPath)
	}
	home, _ := os.UserHomeDir()
	keyPaths = append(keyPaths,
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	)
	for _, kp := range keyPaths {
		if key, err := os.ReadFile(kp); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				break // use first valid key
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH auth methods available (no agent, no key file)")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         p.timeout,
	}

	addr := net.JoinHostPort(hostname, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", host, err)
	}

	p.mu.Lock()
	p.connections[host] = client
	p.mu.Unlock()

	return client, nil
}

// RunCommand executes a command on the given host and returns stdout.
func (p *Pool) RunCommand(host, command string, cmdTimeout int) (string, error) {
	client, err := p.getClient(host)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		// Connection might be stale, discard and retry once
		p.mu.Lock()
		delete(p.connections, host)
		p.mu.Unlock()
		client.Close()

		client, err = p.getClient(host)
		if err != nil {
			return "", err
		}
		session, err = client.NewSession()
		if err != nil {
			return "", fmt.Errorf("SSH session %s: %w", host, err)
		}
	}
	defer session.Close()

	// Run with timeout — use Output() for stdout only, not CombinedOutput
	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = session.Output(command)
		done <- err
	}()

	select {
	case <-time.After(time.Duration(cmdTimeout) * time.Second):
		return "", fmt.Errorf("command timed out after %ds", cmdTimeout)
	case err := <-done:
		if err != nil {
			return string(output), fmt.Errorf("command failed on %s: %w", host, err)
		}
		return string(output), nil
	}
}

// Close closes all cached connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for host, client := range p.connections {
		client.Close()
		delete(p.connections, host)
	}
}
```

- [ ] **Step 6: Implement query.go (high-level query functions)**

```go
// internal/ssh/query.go
package ssh

import (
	"sort"
	"sync"

	"github.com/siyuan/node-monitor/internal/model"
)

// QueryNode queries GPU info from a single node.
func (p *Pool) QueryNode(host string, cmdTimeout int, debug bool) model.NodeStatus {
	output, err := p.RunCommand(host, ListViewCommand(), cmdTimeout)
	if err != nil {
		errMsg := err.Error()
		if !debug {
			// Truncate error for display
			if len(errMsg) > 40 {
				errMsg = errMsg[:40]
			}
		}
		return model.NodeStatus{Hostname: host, Error: &errMsg}
	}

	gpus := parseGPUOutput(output)
	return model.NodeStatus{Hostname: host, GPUs: gpus}
}

// QueryNodeDetail queries detailed GPU, process, and system info from a node.
func (p *Pool) QueryNodeDetail(host string, cmdTimeout int, debug bool) (model.NodeStatus, *model.SystemInfo) {
	output, err := p.RunCommand(host, DetailViewCommand(), cmdTimeout)
	if err != nil {
		errMsg := err.Error()
		if !debug {
			if len(errMsg) > 40 {
				errMsg = errMsg[:40]
			}
		}
		return model.NodeStatus{Hostname: host, Error: &errMsg}, nil
	}

	result := parseDetailOutput(output)
	return model.NodeStatus{Hostname: host, GPUs: result.GPUs}, result.System
}

// QueryAllNodes queries all nodes in parallel, limited by maxWorkers.
func (p *Pool) QueryAllNodes(hosts []string, cmdTimeout int, debug bool, maxWorkers ...int) []model.NodeStatus {
	var wg sync.WaitGroup
	results := make([]model.NodeStatus, len(hosts))

	// Semaphore to limit concurrent SSH connections
	workerLimit := 8
	if len(maxWorkers) > 0 && maxWorkers[0] > 0 {
		workerLimit = maxWorkers[0]
	}
	sem := make(chan struct{}, workerLimit)

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			results[idx] = p.QueryNode(h, cmdTimeout, debug)
		}(i, host)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Hostname < results[j].Hostname
	})

	return results
}
```

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 8: Commit**

```bash
git add internal/ssh/
git commit -m "feat: add SSH connection pool with nvidia-smi parsing"
```

---

## Chunk 3: TUI Components

### Task 6: Styles and GPU Bar Component

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/components/gpubar.go`

- [ ] **Step 1: Implement styles.go**

```go
// internal/tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

// Color thresholds matching the spec.
func UtilColor(percent float64) lipgloss.Color {
	switch {
	case percent < 30:
		return lipgloss.Color("#00FF00") // bright green
	case percent < 60:
		return lipgloss.Color("#FFFF00") // yellow
	case percent < 85:
		return lipgloss.Color("#FF8C00") // orange
	default:
		return lipgloss.Color("#FF0000") // red
	}
}

func MemColor(percent float64) lipgloss.Color {
	switch {
	case percent < 50:
		return lipgloss.Color("#00FFFF") // cyan
	case percent < 75:
		return lipgloss.Color("#FFFF00") // yellow
	case percent < 90:
		return lipgloss.Color("#FF8C00") // orange
	default:
		return lipgloss.Color("#FF0000") // red
	}
}

func NodeStatusIcon(avgUtil float64) string {
	switch {
	case avgUtil > 80:
		return "🔥"
	case avgUtil > 50:
		return "⚡"
	default:
		return "✓"
	}
}

func NodeBorderColor(avgUtil float64) lipgloss.Color {
	switch {
	case avgUtil > 80:
		return lipgloss.Color("#FF0000")
	case avgUtil > 50:
		return lipgloss.Color("#FFFF00")
	default:
		return lipgloss.Color("#00FF00")
	}
}

// Shared styles
var (
	DimStyle    = lipgloss.NewStyle().Faint(true)
	BoldStyle   = lipgloss.NewStyle().Bold(true)
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
)
```

- [ ] **Step 2: Implement gpubar.go**

```go
// internal/tui/components/gpubar.go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
	"github.com/siyuan/node-monitor/internal/tui"
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
	utilColor := tui.UtilColor(float64(gpu.Utilization))
	memColor := tui.MemColor(gpu.MemoryPercent())

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
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI styles and GPU bar component"
```

---

### Task 7: Header Component

**Files:**
- Create: `internal/tui/components/header.go`

- [ ] **Step 1: Implement header.go**

```go
// internal/tui/components/header.go
package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
	"github.com/siyuan/node-monitor/internal/tui"
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

	utilColor := tui.UtilColor(avgUtil)
	avgMemPct := 0.0
	if totalMem > 0 {
		avgMemPct = float64(totalMemUsed) / float64(totalMem) * 100
	}
	memColor := tui.MemColor(avgMemPct)

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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/header.go
git commit -m "feat: add TUI header component"
```

---

### Task 8: Node List Views (Panel + Compact)

**Files:**
- Create: `internal/tui/components/nodelist.go`

- [ ] **Step 1: Implement nodelist.go**

```go
// internal/tui/components/nodelist.go
package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
	"github.com/siyuan/node-monitor/internal/tui"
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

// renderNodePanel renders a single node as a bordered panel with title as first content line.
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
		icon := tui.NodeStatusIcon(avgUtil)
		borderColor = tui.NodeBorderColor(avgUtil)

		titleLine = lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(icon + " " + node.Hostname)

		for _, gpu := range node.GPUs {
			contentLines = append(contentLines, RenderGPURow(gpu, 15))
		}

		// Summary
		utilColor := tui.UtilColor(avgUtil)
		memPct := 0.0
		if node.TotalMemory() > 0 {
			memPct = float64(node.TotalMemoryUsed()) / float64(node.TotalMemory()) * 100
		}
		memColor := tui.MemColor(memPct)

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

		icon := tui.NodeStatusIcon(node.AvgUtilization())

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

				utilColor1 := tui.UtilColor(float64(gpu1.Utilization))
				bar1 := RenderBar(float64(gpu1.Utilization), barWidth, utilColor1)
				pct1 := lipgloss.NewStyle().Bold(true).Foreground(utilColor1).Render(fmt.Sprintf("%3d%%", gpu1.Utilization))
				mem1 := fmt.Sprintf("%s/%s", model.FormatMemory(gpu1.MemoryUsed), model.FormatMemory(gpu1.MemoryTotal))

				right := ""
				if i+1 < len(node.GPUs) {
					gpu2 := node.GPUs[i+1]
					utilColor2 := tui.UtilColor(float64(gpu2.Utilization))
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

				utilColor := tui.UtilColor(float64(gpu.Utilization))
				memColor := tui.MemColor(gpu.MemoryPercent())
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

	// Sort users for deterministic output
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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/nodelist.go
git commit -m "feat: add panel and compact node list views"
```

---

### Task 9: Node Detail View

**Files:**
- Create: `internal/tui/components/nodedetail.go`

- [ ] **Step 1: Implement nodedetail.go**

```go
// internal/tui/components/nodedetail.go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
	"github.com/siyuan/node-monitor/internal/tui"
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
	borderColor := tui.NodeBorderColor(node.AvgUtilization())
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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/nodedetail.go
git commit -m "feat: add node detail drill-down view"
```

---

### Task 10: Help Overlay

**Files:**
- Create: `internal/tui/components/help.go`

- [ ] **Step 1: Implement help.go**

```go
// internal/tui/components/help.go
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpEntries = []struct {
	key  string
	desc string
}{
	{"↑/↓, j/k", "Navigate nodes"},
	{"Enter", "Node detail view"},
	{"Esc, q", "Back / Exit"},
	{"Tab", "Toggle panel/compact"},
	{"p", "Toggle processes"},
	{"s", "Cycle sort order"},
	{"g", "Cycle node group"},
	{"/", "Search nodes"},
	{"?", "Toggle this help"},
}

// RenderHelp renders the keyboard shortcut help overlay.
func RenderHelp(width, height int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, e := range helpEntries {
		key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Width(12).Render(e.key)
		lines = append(lines, "  "+key+"  "+e.desc)
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(1, 3).
		Width(40).
		Align(lipgloss.Left)

	rendered := style.Render(content)

	// Center the overlay
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, rendered)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/help.go
git commit -m "feat: add help overlay component"
```

---

## Chunk 4: Main TUI App, CLI Integration, Static Mode

### Task 11: Bubble Tea Main App

**Files:**
- Create: `internal/tui/app.go`

- [ ] **Step 1: Implement app.go — the Bubble Tea Model/Update/View**

```go
// internal/tui/app.go
package tui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siyuan/node-monitor/internal/model"
	sshpool "github.com/siyuan/node-monitor/internal/ssh"
	"github.com/siyuan/node-monitor/internal/tui/components"
)

// View modes
type ViewMode int

const (
	ViewPanel ViewMode = iota
	ViewCompact
)

// App state
type AppView int

const (
	AppList AppView = iota
	AppDetail
)

// Messages
type tickMsg time.Time
type nodesUpdatedMsg []model.NodeStatus
type detailResultMsg struct {
	node   model.NodeStatus
	system *model.SystemInfo
}

// Sort modes
type SortMode int

const (
	SortName SortMode = iota
	SortUtil
	SortMemory
)

// Model is the Bubble Tea model.
type Model struct {
	// Data
	nodes       []model.NodeStatus
	detailNode  *model.NodeStatus
	detailSys   *model.SystemInfo

	// Config
	allHosts      []string // original full host list (never modified)
	hosts         []string // current active host list (may be filtered by group)
	pool          *sshpool.Pool
	interval      time.Duration
	cmdTimeout    int
	debug         bool
	showProcesses bool

	// UI state
	viewMode    ViewMode
	appView     AppView
	selectedIdx int
	sortMode    SortMode
	showHelp    bool
	searchQuery string
	searching   bool
	width       int
	height      int

	// Groups
	groups       map[string][]string
	groupNames   []string
	currentGroup int // -1 = all
}

// NewModel creates a new TUI model.
func NewModel(
	hosts []string,
	pool *sshpool.Pool,
	interval float64,
	cmdTimeout int,
	debug bool,
	showProcesses bool,
	viewMode ViewMode,
	groups map[string][]string,
) Model {
	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	return Model{
		allHosts:      hosts,
		hosts:         hosts,
		pool:          pool,
		interval:      time.Duration(interval * float64(time.Second)),
		cmdTimeout:    cmdTimeout,
		debug:         debug,
		showProcesses: showProcesses,
		viewMode:      viewMode,
		appView:       AppList,
		selectedIdx:   0,
		currentGroup:  -1,
		groups:        groups,
		groupNames:    groupNames,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.queryNodes(),
		m.tick(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(m.queryNodes(), m.tick())

	case nodesUpdatedMsg:
		m.nodes = sortNodes([]model.NodeStatus(msg), m.sortMode)
		// Clamp selected index
		if m.selectedIdx >= len(m.nodes) {
			m.selectedIdx = len(m.nodes) - 1
		}
		if m.selectedIdx < 0 {
			m.selectedIdx = 0
		}
		return m, nil

	case detailResultMsg:
		n := msg.node
		m.detailNode = &n
		m.detailSys = msg.system
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var view string

	header := components.RenderHeader(m.nodes, m.interval.Seconds(), m.width)

	switch m.appView {
	case AppList:
		var content string
		switch m.viewMode {
		case ViewPanel:
			content = components.RenderPanelView(m.nodes, m.selectedIdx, m.width)
		case ViewCompact:
			content = components.RenderCompactView(m.nodes, m.selectedIdx, m.width, m.showProcesses)
		}

		footer := ""
		if m.searching {
			footer = "\n  Search: " + m.searchQuery + "█"
		}

		view = header + "\n\n" + content + footer

	case AppDetail:
		if m.detailNode != nil {
			view = header + "\n\n" + components.RenderNodeDetail(*m.detailNode, m.detailSys, m.width)
		} else {
			view = header + "\n\n  Loading detail..."
		}
	}

	if m.showHelp {
		return components.RenderHelp(m.width, m.height)
	}

	return view
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search mode handles keys differently
	if m.searching {
		switch msg.Type {
		case tea.KeyEscape:
			m.searching = false
			m.searchQuery = ""
			return m, nil
		case tea.KeyEnter:
			m.searching = false
			return m, nil
		case tea.KeyBackspace:
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.searchQuery += msg.String()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		if m.appView == AppDetail {
			m.appView = AppList
			m.detailNode = nil
			m.detailSys = nil
			return m, nil
		}
		return m, tea.Quit

	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.appView == AppDetail {
			m.appView = AppList
			m.detailNode = nil
			m.detailSys = nil
			return m, nil
		}
		return m, tea.Quit

	case "up", "k":
		if m.appView == AppList && m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "down", "j":
		if m.appView == AppList && m.selectedIdx < len(m.nodes)-1 {
			m.selectedIdx++
		}
		return m, nil

	case "enter":
		if m.appView == AppList && len(m.nodes) > 0 && m.selectedIdx < len(m.nodes) {
			node := m.nodes[m.selectedIdx]
			if node.IsOnline() {
				m.appView = AppDetail
				m.detailNode = nil
				m.detailSys = nil
				return m, m.queryDetail(node.Hostname)
			}
		}
		return m, nil

	case "tab":
		if m.viewMode == ViewPanel {
			m.viewMode = ViewCompact
		} else {
			m.viewMode = ViewPanel
		}
		return m, nil

	case "p":
		m.showProcesses = !m.showProcesses
		return m, nil

	case "s":
		m.sortMode = (m.sortMode + 1) % 3
		m.nodes = sortNodes(m.nodes, m.sortMode)
		return m, nil

	case "g":
		if len(m.groupNames) > 0 {
			m.currentGroup++
			if m.currentGroup >= len(m.groupNames) {
				m.currentGroup = -1 // back to "all"
			}
			// Update hosts based on group, restoring allHosts for "all"
			if m.currentGroup >= 0 {
				m.hosts = m.groups[m.groupNames[m.currentGroup]]
			} else {
				m.hosts = m.allHosts
			}
			// Reset selection
			m.selectedIdx = 0
		}
		return m, nil

	case "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil

	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	return m, nil
}

// Commands

func (m Model) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) queryNodes() tea.Cmd {
	return func() tea.Msg {
		hosts := m.hosts
		if m.searching && m.searchQuery != "" {
			var filtered []string
			for _, h := range hosts {
				if strings.Contains(h, m.searchQuery) {
					filtered = append(filtered, h)
				}
			}
			hosts = filtered
		}
		results := m.pool.QueryAllNodes(hosts, m.cmdTimeout, m.debug, len(hosts))
		return nodesUpdatedMsg(results)
	}
}

func (m Model) queryDetail(hostname string) tea.Cmd {
	return func() tea.Msg {
		node, sys := m.pool.QueryNodeDetail(hostname, m.cmdTimeout, m.debug)
		return detailResultMsg{node: node, system: sys}
	}
}

func sortNodes(nodes []model.NodeStatus, mode SortMode) []model.NodeStatus {
	sorted := make([]model.NodeStatus, len(nodes))
	copy(sorted, nodes)

	switch mode {
	case SortName:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Hostname < sorted[j].Hostname
		})
	case SortUtil:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AvgUtilization() > sorted[j].AvgUtilization()
		})
	case SortMemory:
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].TotalMemoryUsed() > sorted[j].TotalMemoryUsed()
		})
	}

	return sorted
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add Bubble Tea main app with keyboard navigation"
```

---

### Task 12: CLI Integration

**Files:**
- Modify: `cmd/root.go`
- Modify: `main.go`

- [ ] **Step 1: Update cmd/root.go with all flags and app startup**

```go
// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/siyuan/node-monitor/internal/config"
	"github.com/siyuan/node-monitor/internal/slurm"
	sshpool "github.com/siyuan/node-monitor/internal/ssh"
	"github.com/siyuan/node-monitor/internal/tui"
	"github.com/siyuan/node-monitor/internal/tui/components"
)

var appVersion string

var (
	flagNodes     string
	flagGroup     string
	flagInterval  float64
	flagWorkers   int
	flagCompact   bool
	flagStatic    bool
	flagProcesses bool
	flagDebug     bool
)

var rootCmd = &cobra.Command{
	Use:   "node-monitor",
	Short: "GPU Cluster Monitor - Monitor GPU resources across Slurm nodes",
	Long: `A beautiful, interactive terminal dashboard for monitoring GPU utilization
and memory usage across Slurm cluster nodes in real-time.

Examples:
  node-monitor                        Auto-detect Slurm nodes
  node-monitor -n visko-1,visko-2     Monitor specific nodes
  node-monitor -c                     Compact table view
  node-monitor -c -p                  Compact + show processes
  node-monitor -s                     Print once and exit
  node-monitor --group train          Monitor a node group`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&flagNodes, "nodes", "n", "", "Comma-separated list of nodes")
	rootCmd.Flags().StringVarP(&flagGroup, "group", "g", "", "Node group from config file")
	rootCmd.Flags().Float64VarP(&flagInterval, "interval", "i", 0, "Refresh interval in seconds")
	rootCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 0, "Max parallel SSH connections")
	rootCmd.Flags().BoolVarP(&flagCompact, "compact", "c", false, "Start in compact view mode")
	rootCmd.Flags().BoolVarP(&flagStatic, "static", "s", false, "Print once and exit (no TUI)")
	rootCmd.Flags().BoolVarP(&flagProcesses, "processes", "p", false, "Show GPU processes")
	rootCmd.Flags().BoolVarP(&flagDebug, "debug", "d", false, "Verbose SSH error output")
}

func run(cmd *cobra.Command, args []string) error {
	// Load config file
	cfg := config.Load()

	// CLI flags override config
	if cmd.Flags().Changed("interval") {
		cfg.Interval = flagInterval
	}
	if cmd.Flags().Changed("workers") {
		cfg.Workers = flagWorkers
	}
	if flagCompact {
		cfg.View = "compact"
	}
	if flagProcesses {
		cfg.Processes = true
	}
	if flagDebug {
		cfg.Debug = true
	}
	if flagStatic {
		cfg.Static = true
	}

	// Resolve nodes
	var hosts []string
	if flagNodes != "" {
		hosts = slurm.ParseNodeList(flagNodes)
	} else {
		hosts = cfg.ResolveNodes(flagGroup)
	}

	// If no nodes configured, try Slurm auto-detection
	if len(hosts) == 0 {
		fmt.Println("🔍 Detecting Slurm nodes...")
		detected, err := slurm.DetectNodes()
		if err != nil {
			return fmt.Errorf("no nodes specified and Slurm detection failed: %w\nUse --nodes to specify nodes manually", err)
		}
		hosts = detected
		fmt.Printf("✓ Found %d nodes: %v\n", len(hosts), hosts)
	}

	// Create SSH pool
	pool := sshpool.NewPool(cfg.SSH.ConnectTimeout, cfg.SSH.User, cfg.SSH.IdentityFile)
	defer pool.Close()

	// Static mode: query once and print
	if cfg.Static {
		results := pool.QueryAllNodes(hosts, cfg.SSH.CommandTimeout, cfg.Debug, cfg.Workers)

		// Auto-detect terminal width, fallback to 120
		termWidth := 120
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			termWidth = w
		}

		header := components.RenderHeader(results, cfg.Interval, termWidth)
		fmt.Println(header)
		fmt.Println()

		viewMode := tui.ViewPanel
		if cfg.View == "compact" {
			viewMode = tui.ViewCompact
		}
		if viewMode == tui.ViewCompact {
			fmt.Println(components.RenderCompactView(results, -1, termWidth, cfg.Processes))
		} else {
			fmt.Println(components.RenderPanelView(results, -1, termWidth))
		}
		return nil
	}

	// Dashboard mode
	viewMode := tui.ViewPanel
	if cfg.View == "compact" {
		viewMode = tui.ViewCompact
	}

	model := tui.NewModel(
		hosts,
		pool,
		cfg.Interval,
		cfg.SSH.CommandTimeout,
		cfg.Debug,
		cfg.Processes,
		viewMode,
		cfg.Groups,
	)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify full build and run**

Run: `go build -o node-monitor . && ./node-monitor --version`
Expected: Prints version string.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go main.go
git commit -m "feat: integrate CLI with TUI and SSH pool"
```

---

### Task 13: Static Mode Output

Static mode is already handled in Task 12's `cmd/root.go` (the `cfg.Static` branch). No separate task needed — it renders components without Bubble Tea.

---

## Chunk 5: Build, Release, Polish

### Task 14: Makefile and GoReleaser

**Files:**
- Create: `Makefile`
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Create Makefile**

```makefile
# Makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build clean test

build:
	go build -ldflags "$(LDFLAGS)" -o node-monitor .

test:
	go test ./... -v

clean:
	rm -f node-monitor

install: build
	cp node-monitor ~/.local/bin/node-monitor
```

- [ ] **Step 2: Create .goreleaser.yaml**

```yaml
# .goreleaser.yaml
version: 2

builds:
  - binary: node-monitor
    main: .
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: siyuan
    name: node-monitor
```

- [ ] **Step 3: Test build with Makefile**

Run: `make build && ./node-monitor --version`
Expected: Binary built with version string.

- [ ] **Step 4: Commit**

```bash
git add Makefile .goreleaser.yaml
git commit -m "feat: add Makefile and GoReleaser config"
```

---

### Task 15: Run All Tests and Final Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass.

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues.

- [ ] **Step 3: Test build**

Run: `make build`
Expected: Binary built successfully.

- [ ] **Step 4: Manual smoke test**

Run: `./node-monitor --version`
Expected: Prints version.

Run: `./node-monitor -n localhost -s` (if SSH to localhost works)
Expected: Prints GPU info or error message.

- [ ] **Step 5: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "chore: fix issues found during final verification"
```

---

## Summary

| Chunk | Tasks | Description |
|-------|-------|-------------|
| 1 | 1-3 | Scaffolding, data model, config system |
| 2 | 4-5 | Slurm detection, SSH pool, nvidia-smi parsing |
| 3 | 6-10 | TUI components (styles, bars, header, views, detail, help) |
| 4 | 11-12 | Bubble Tea app, CLI integration |
| 5 | 14-15 | Build system, final verification |

**Total: 15 tasks, ~70 steps**

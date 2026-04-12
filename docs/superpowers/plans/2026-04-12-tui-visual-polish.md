# TUI Visual Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refine the TUI to btop++ aesthetic quality with muted colors, inline process display, smart hostname truncation, and fixed card alignment.

**Architecture:** Update palette constants in `styles.go`, change bar chars in `gpubar.go`, add hostname logic and process rendering to `nodecard.go`, update `app.go` for new expand/collapse behavior, and propagate palette changes to all remaining components.

**Tech Stack:** Go, Bubble Tea, Lip Gloss. Tests via `go test`.

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `internal/tui/components/styles.go` | Color palette, gradient definitions | Update all constants + gradient RGB |
| `internal/tui/components/gpubar.go` | Progress bars, heatmap | Change fill char `■` → `█` |
| `internal/tui/components/hostname.go` | Smart hostname display logic | **Create** |
| `internal/tui/components/hostname_test.go` | Tests for hostname truncation | **Create** |
| `internal/tui/components/nodecard.go` | Card rendering with process lines | Rewrite: add processes, fix alignment |
| `internal/tui/components/header.go` | Outer frame, divider rendering | Fix divider connection |
| `internal/tui/components/nodedetail.go` | Detail panel rendering | Palette update |
| `internal/tui/components/proctable.go` | Process table (used by static mode) | Palette update |
| `internal/tui/components/help.go` | Help overlay | Palette update |
| `internal/tui/app.go` | Main TUI model, update/view loop | Remove bottom process panel, adjust `p` key, variable card height mouse regions |
| `cmd/root.go` | CLI entry point | Remove `--processes` flag, palette in static mode |

---

### Task 1: Update Color Palette

**Files:**
- Modify: `internal/tui/components/styles.go`

- [ ] **Step 1: Update all color constants**

Replace the entire color/gradient section in `styles.go`:

```go
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// btop++-inspired color palette (muted, layered)
var (
	ColorFg        = lipgloss.Color("#c9d1d9")
	ColorDim       = lipgloss.Color("#484f58")
	ColorBorder    = lipgloss.Color("#30363d")
	ColorMeterBg   = lipgloss.Color("#21262d")
	ColorAccent    = lipgloss.Color("#388bfd")
	ColorGreen     = lipgloss.Color("#2ea043")
	ColorYellow    = lipgloss.Color("#d29922")
	ColorOrange    = lipgloss.Color("#cf6e2a")
	ColorRed       = lipgloss.Color("#da3633")
	ColorSelection = lipgloss.Color("#e6edf3")
)

// Gradient defines a 3-color interpolation (start → mid → end).
type Gradient struct {
	Start [3]int
	Mid   [3]int
	End   [3]int
}

var (
	UtilGradient = Gradient{
		Start: [3]int{0x2e, 0xa0, 0x43}, // green
		Mid:   [3]int{0xd2, 0x99, 0x22}, // amber
		End:   [3]int{0xda, 0x36, 0x33}, // red
	}
	MemGradient = Gradient{
		Start: [3]int{0x38, 0x8b, 0xfd}, // blue
		Mid:   [3]int{0xd2, 0x99, 0x22}, // amber
		End:   [3]int{0xda, 0x36, 0x33}, // red
	}
)

// ColorAt returns an interpolated hex color for a value 0-100.
func (g Gradient) ColorAt(pct int) lipgloss.Color {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	var r, gn, b int
	if pct <= 50 {
		t := float64(pct) / 50.0
		r = g.Start[0] + int(t*float64(g.Mid[0]-g.Start[0]))
		gn = g.Start[1] + int(t*float64(g.Mid[1]-g.Start[1]))
		b = g.Start[2] + int(t*float64(g.Mid[2]-g.Start[2]))
	} else {
		t := float64(pct-50) / 50.0
		r = g.Mid[0] + int(t*float64(g.End[0]-g.Mid[0]))
		gn = g.Mid[1] + int(t*float64(g.End[1]-g.Mid[1]))
		b = g.Mid[2] + int(t*float64(g.End[2]-g.Mid[2]))
	}

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, gn, b))
}

// NodeStatusIcon returns an icon based on average utilization.
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

// NodeBorderColor returns border color based on utilization.
func NodeBorderColor(avgUtil float64) lipgloss.Color {
	switch {
	case avgUtil > 80:
		return ColorRed
	case avgUtil > 50:
		return ColorYellow
	default:
		return ColorGreen
	}
}

// UtilColor returns a single color for a utilization percentage.
func UtilColor(percent float64) lipgloss.Color {
	return UtilGradient.ColorAt(int(percent))
}

// MemColor returns a single color for a memory percentage.
func MemColor(percent float64) lipgloss.Color {
	return MemGradient.ColorAt(int(percent))
}
```

- [ ] **Step 2: Build to verify no compile errors**

Run: `go build ./...`
Expected: success, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/styles.go
git commit -m "style: update color palette from Dracula to btop++ muted tones"
```

---

### Task 2: Update Bar Characters

**Files:**
- Modify: `internal/tui/components/gpubar.go`

- [ ] **Step 1: Change fill character from ■ to █**

In `RenderGradientBar`, change the filled character:

```go
// In the filled branch (i < filled):
b.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
```

The unfilled character `░` stays the same — it already uses `ColorMeterBg` which is now `#21262d`.

- [ ] **Step 2: Build to verify**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/gpubar.go
git commit -m "style: use full-block █ for progress bar fills"
```

---

### Task 3: Smart Hostname Truncation

**Files:**
- Create: `internal/tui/components/hostname.go`
- Create: `internal/tui/components/hostname_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/components/hostname_test.go`:

```go
package components

import "testing"

func TestComputeDisplayNames_CommonPrefix(t *testing.T) {
	hosts := []string{
		"host-10-240-99-115",
		"host-10-240-99-116",
		"host-10-240-99-117",
		"host-10-240-99-118",
		"host-10-240-99-119",
		"host-10-240-99-120",
	}
	names := ComputeDisplayNames(hosts)
	expected := []string{"115", "116", "117", "118", "119", "120"}
	for i, want := range expected {
		if names[hosts[i]] != want {
			t.Errorf("host %q: got %q, want %q", hosts[i], names[hosts[i]], want)
		}
	}
}

func TestComputeDisplayNames_SingleNode(t *testing.T) {
	hosts := []string{"gpu-server-01"}
	names := ComputeDisplayNames(hosts)
	if names["gpu-server-01"] != "gpu-server-01" {
		t.Errorf("single node: got %q, want full hostname", names["gpu-server-01"])
	}
}

func TestComputeDisplayNames_NothingInCommon(t *testing.T) {
	hosts := []string{"alpha", "beta", "gamma"}
	names := ComputeDisplayNames(hosts)
	if names["alpha"] != "alpha" {
		t.Errorf("no common prefix: got %q, want %q", names["alpha"], "alpha")
	}
}

func TestComputeDisplayNames_PartialPrefix(t *testing.T) {
	hosts := []string{"node-a1", "node-a2", "node-b1"}
	names := ComputeDisplayNames(hosts)
	// Common prefix is "node-", which ends at a boundary
	expected := map[string]string{"node-a1": "a1", "node-a2": "a2", "node-b1": "b1"}
	for h, want := range expected {
		if names[h] != want {
			t.Errorf("host %q: got %q, want %q", h, names[h], want)
		}
	}
}

func TestComputeDisplayNames_EmptyAfterStrip(t *testing.T) {
	// All identical hostnames — stripping leaves empty, so keep full
	hosts := []string{"same", "same"}
	names := ComputeDisplayNames(hosts)
	if names["same"] != "same" {
		t.Errorf("identical hosts: got %q, want full hostname", names["same"])
	}
}

func TestTruncateHostname(t *testing.T) {
	if got := TruncateHostname("very-long-hostname-that-wont-fit", 10); got != "..wont-fit" {
		t.Errorf("got %q, want %q", got, "..wont-fit")
	}
	if got := TruncateHostname("short", 10); got != "short" {
		t.Errorf("got %q, want %q", got, "short")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/components/ -run TestComputeDisplayNames -v`
Expected: FAIL — function not defined.

- [ ] **Step 3: Implement hostname logic**

Create `internal/tui/components/hostname.go`:

```go
package components

import "strings"

// ComputeDisplayNames computes short display names for a list of hostnames.
// It strips the longest common prefix (at a natural boundary) and returns
// a map from full hostname to display name.
func ComputeDisplayNames(hosts []string) map[string]string {
	result := make(map[string]string, len(hosts))

	if len(hosts) <= 1 {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	// Find longest common prefix
	prefix := hosts[0]
	for _, h := range hosts[1:] {
		for !strings.HasPrefix(h, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				break
			}
		}
		if prefix == "" {
			break
		}
	}

	// Trim prefix to natural boundary (last '.', '-', or '_')
	trimmed := prefix
	for len(trimmed) > 0 {
		last := trimmed[len(trimmed)-1]
		if last == '.' || last == '-' || last == '_' {
			break
		}
		trimmed = trimmed[:len(trimmed)-1]
	}

	// Validate: stripping must leave non-empty, distinct suffixes
	if trimmed == "" {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	allNonEmpty := true
	for _, h := range hosts {
		suffix := strings.TrimPrefix(h, trimmed)
		if suffix == "" {
			allNonEmpty = false
			break
		}
	}

	if !allNonEmpty {
		for _, h := range hosts {
			result[h] = h
		}
		return result
	}

	for _, h := range hosts {
		result[h] = strings.TrimPrefix(h, trimmed)
	}
	return result
}

// TruncateHostname truncates a display name to maxLen, adding ".." prefix if needed.
func TruncateHostname(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	if maxLen <= 2 {
		return name[:maxLen]
	}
	return ".." + name[len(name)-(maxLen-2):]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/components/ -run "TestComputeDisplayNames|TestTruncateHostname" -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/components/hostname.go internal/tui/components/hostname_test.go
git commit -m "feat: add smart hostname truncation with common prefix stripping"
```

---

### Task 4: Rewrite Node Card Rendering

**Files:**
- Modify: `internal/tui/components/nodecard.go`

This is the largest task. The card now shows inline processes, uses short hostnames, and fixes the `embedTitle` alignment bug.

- [ ] **Step 1: Rewrite nodecard.go**

Replace the entire file content of `internal/tui/components/nodecard.go`:

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siyuan/node-monitor/internal/model"
)

// RenderNodeGrid renders all nodes as a grid of condensed cards.
func RenderNodeGrid(nodes []model.NodeStatus, selectedIdx int, width int, displayNames map[string]string, expanded bool) string {
	minCardWidth := 40
	numCols := width / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	if numCols > len(nodes) && len(nodes) > 0 {
		numCols = len(nodes)
	}
	cardWidth := width/numCols - 1
	if cardWidth < minCardWidth {
		cardWidth = minCardWidth
	}

	var cards []string
	for i, node := range nodes {
		name := node.Hostname
		if dn, ok := displayNames[node.Hostname]; ok {
			name = dn
		}
		cards = append(cards, renderCondensedCard(node, cardWidth, i == selectedIdx, name, expanded))
	}

	// Arrange in rows
	var rows []string
	for i := 0; i < len(cards); i += numCols {
		end := i + numCols
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

// renderCondensedCard renders a compact node card with optional process lines.
func renderCondensedCard(node model.NodeStatus, width int, selected bool, displayName string, expanded bool) string {
	innerWidth := width - 4 // border + padding
	barWidth := innerWidth - 12
	if barWidth < 10 {
		barWidth = 10
	}

	borderColor := ColorBorder

	var titleIcon string
	var contentLines []string

	if !node.IsOnline() {
		titleIcon = "✗"
		borderColor = ColorRed

		errMsg := "Offline"
		if node.Error != nil && len(*node.Error) > 0 {
			errMsg = *node.Error
			if len(errMsg) > innerWidth-4 {
				errMsg = errMsg[:innerWidth-4]
			}
		}
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorRed).Render(" ⚠ "+errMsg),
			"",
			"",
		)
	} else if len(node.GPUs) == 0 {
		titleIcon = "?"
		borderColor = ColorYellow
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorDim).Render(" No GPUs detected"),
			"",
			"",
		)
	} else {
		avgUtil := node.AvgUtilization()
		titleIcon = NodeStatusIcon(avgUtil)
		borderColor = NodeBorderColor(avgUtil)

		// Line 1: Aggregate utilization bar
		utilBar := RenderGradientBar(avgUtil, barWidth, UtilGradient)
		utilPct := lipgloss.NewStyle().Bold(true).Foreground(UtilColor(avgUtil)).Render(fmt.Sprintf("%3.0f%%", avgUtil))
		utilLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("Util ")
		contentLines = append(contentLines, utilLabel+utilBar+" "+utilPct)

		// Line 2: Aggregate memory bar
		memPct := 0.0
		if node.TotalMemory() > 0 {
			memPct = float64(node.TotalMemoryUsed()) / float64(node.TotalMemory()) * 100
		}
		memBar := RenderGradientBar(memPct, barWidth, MemGradient)
		memVal := lipgloss.NewStyle().Bold(true).Foreground(MemColor(memPct)).Render(model.FormatMemory(node.TotalMemoryUsed()))
		memLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("Mem  ")
		contentLines = append(contentLines, memLabel+memBar+" "+memVal)

		// Line 3: GPU heatmap + model + idle indicator
		heatmap := RenderGPUHeatmap(node.GPUs)
		gpuLabel := lipgloss.NewStyle().Foreground(ColorDim).Render("GPU  ")
		modelStr := lipgloss.NewStyle().Foreground(ColorDim).Render(" " + node.GPUModelSummary())

		users := node.ActiveUsers()
		if len(users) == 0 {
			idleStr := lipgloss.NewStyle().Foreground(ColorBorder).Italic(true).Render("  idle")
			contentLines = append(contentLines, gpuLabel+heatmap+modelStr+idleStr)
		} else {
			contentLines = append(contentLines, gpuLabel+heatmap+modelStr)
		}

		// Process lines (only when expanded and there are processes)
		if expanded && len(users) > 0 {
			procLines := renderCardProcesses(node, innerWidth)
			contentLines = append(contentLines, procLines...)
		}
	}

	content := strings.Join(contentLines, "\n")

	// Build card with selection override
	titleColor := borderColor
	if selected {
		borderColor = ColorSelection
		titleColor = ColorSelection
	}

	titleText := lipgloss.NewStyle().Bold(true).Foreground(titleColor).Render(titleIcon + " " + displayName)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	rendered := style.Render(content)

	// Replace top border with embedded title
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		lines[0] = embedTitle(lines[0], titleText, borderColor, width)
	}

	return strings.Join(lines, "\n")
}

// renderCardProcesses renders per-user process lines for a card.
func renderCardProcesses(node model.NodeStatus, innerWidth int) []string {
	type userAgg struct {
		User string
		GPUs []int
		Mem  int
		Cmd  string
	}

	aggMap := make(map[string]*userAgg)
	var userOrder []string
	for _, g := range node.GPUs {
		for _, p := range g.Processes {
			if a, ok := aggMap[p.User]; ok {
				a.GPUs = appendUniqueInt(a.GPUs, p.GPUIndex)
				a.Mem += p.MemoryMiB
			} else {
				aggMap[p.User] = &userAgg{
					User: p.User,
					GPUs: []int{p.GPUIndex},
					Mem:  p.MemoryMiB,
					Cmd:  p.Command,
				}
				userOrder = append(userOrder, p.User)
			}
		}
	}

	var lines []string
	for _, user := range userOrder {
		a := aggMap[user]
		sortInts(a.GPUs)
		gpuStr := formatGPURange(a.GPUs)

		// Command: basename only, truncate
		cmd := baseName(a.Cmd)
		maxCmd := innerWidth - 24
		if maxCmd < 6 {
			maxCmd = 6
		}
		if len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-2] + ".."
		}

		line := lipgloss.NewStyle().Foreground(ColorAccent).Render(fmt.Sprintf("%-5s", truncStr(user, 5))) + " " +
			lipgloss.NewStyle().Foreground(ColorGreen).Render(fmt.Sprintf("%-3s", gpuStr)) + " " +
			lipgloss.NewStyle().Foreground(ColorFg).Render(fmt.Sprintf("%4s", model.FormatMemory(a.Mem))) + " " +
			lipgloss.NewStyle().Foreground(ColorDim).Render(cmd)
		lines = append(lines, line)
	}
	return lines
}

// embedTitle replaces the top border line with ╭─┤ title ├───╮.
// Uses rune counting to correctly handle multi-byte characters.
func embedTitle(topLine, title string, borderColor lipgloss.Color, targetWidth int) string {
	bc := func(s string) string {
		return lipgloss.NewStyle().Foreground(borderColor).Render(s)
	}

	prefix := bc("╭─┤ ")
	suffix := bc(" ├")

	titleRendered := prefix + title + suffix
	titleVisualWidth := lipgloss.Width(titleRendered)

	// Fill remaining width, accounting for closing ╮
	remaining := targetWidth - titleVisualWidth - 1
	if remaining < 0 {
		remaining = 0
	}

	return titleRendered + bc(strings.Repeat("─", remaining)) + bc("╮")
}

// Helper functions

func appendUniqueInt(slice []int, val int) []int {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func baseName(cmd string) string {
	// Strip path: "/usr/bin/python train.py" -> "python train.py"
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}
	exe := parts[0]
	idx := strings.LastIndex(exe, "/")
	if idx >= 0 {
		exe = exe[idx+1:]
	}
	parts[0] = exe
	return strings.Join(parts, " ")
}

// formatGPURange formats GPU indices compactly: [0,1,2,3] -> "0-7", [0,2,5] -> "0,2,5"
func formatGPURange(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return fmt.Sprintf("%d", ids[0])
	}
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
```

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./...`
Expected: FAIL — `RenderNodeGrid` signature changed (now takes `displayNames` and `expanded`). That's expected; we'll fix callers in Task 6.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/nodecard.go
git commit -m "feat: rewrite node cards with inline processes and fixed alignment"
```

---

### Task 5: Fix Outer Frame Divider Connection

**Files:**
- Modify: `internal/tui/components/header.go`

- [ ] **Step 1: Update RenderDivider to use proper box-drawing connectors**

The `RenderDivider` function already uses `├` and `┤`. Verify it renders at the correct width by ensuring `buildDivider` in `app.go` delegates to it properly. No code change needed in `header.go` for this — the divider logic is correct; the issue is in `app.go`'s `buildDivider` which bypasses `RenderDivider`. We'll fix that in Task 6.

- [ ] **Step 2: Update palette colors in header.go**

No color constant names changed — they're all imported from `styles.go`. Since the variable names (`ColorBorder`, `ColorFg`, etc.) are unchanged, `header.go` automatically picks up the new palette. No changes needed.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: builds (may still fail due to Task 4's signature change — that's OK).

---

### Task 6: Update app.go for New Behavior

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add displayNames and expanded fields to Model, update constructor**

Update the `Model` struct and `NewModel`:

```go
// Add to Model struct fields, in the "UI state" section:
	expanded     bool              // cards show inline processes
	displayNames map[string]string // hostname -> short display name

// In NewModel, before the return statement, add:
	displayNames := ComputeDisplayNames(hosts)

// Update the return to include:
	return Model{
		allHosts:      hosts,
		hosts:         hosts,
		pool:          pool,
		interval:      time.Duration(interval * float64(time.Second)),
		cmdTimeout:    cmdTimeout,
		debug:         debug,
		showProcesses: true,
		expanded:      true,
		displayNames:  displayNames,
		viewMode:      viewMode,
		bottomPanel:   PanelProcesses,
		selectedIdx:   0,
		currentGroup:  -1,
		groups:        groups,
		groupNames:    groupNames,
	}
```

Add the import for `components`:

```go
import (
	...
	"github.com/siyuan/node-monitor/internal/tui/components"
)
```

Note: `components` is already imported. `ComputeDisplayNames` must be called as `components.ComputeDisplayNames`.

- [ ] **Step 2: Update View() to pass new args and remove bottom process panel**

Replace the `View()` method body:

```go
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return components.RenderHelp(m.width, m.height)
	}

	innerWidth := m.width - 2 // outer frame borders

	// Header
	header := components.RenderHeader(m.nodes, m.interval.Seconds(), innerWidth)

	// Node grid with display names and expanded state
	nodeGrid := components.RenderNodeGrid(m.nodes, m.selectedIdx, innerWidth, m.displayNames, m.expanded)

	// Bottom panel — only for detail view now
	var bottomTitle string
	var bottomContent string

	if m.bottomPanel == PanelDetail && m.detailNode != nil {
		bottomTitle = m.detailNode.Hostname + " Detail"
		bottomContent = components.RenderNodeDetail(*m.detailNode, m.detailSys, innerWidth)
	}

	// Assemble body
	var bodyLines []string

	for _, line := range strings.Split(nodeGrid, "\n") {
		bodyLines = append(bodyLines, " "+line)
	}

	// Divider + bottom panel (only when detail view is open)
	if bottomContent != "" {
		bodyLines = append(bodyLines, "")
		divLine := components.RenderDivider(bottomTitle, m.width)
		bodyLines = append(bodyLines, divLine)
		for _, line := range strings.Split(bottomContent, "\n") {
			bodyLines = append(bodyLines, " "+line)
		}
	}

	// Search bar
	if m.searching {
		searchLine := " Search: " + m.searchQuery + "█"
		bodyLines = append(bodyLines, searchLine)
	}

	body := strings.Join(bodyLines, "\n")

	return components.RenderOuterFrame(header, body, m.width, m.height)
}
```

- [ ] **Step 3: Update `p` key handler to toggle expanded instead of showProcesses**

In `handleKey`, change the `"p"` case:

```go
	case "p":
		m.expanded = !m.expanded
		return m, nil
```

- [ ] **Step 4: Remove the buildDivider helper function from app.go**

Delete the `buildDivider` function (lines ~248-258 in the original) — we now use `components.RenderDivider` directly.

- [ ] **Step 5: Update mouse handler for variable card heights**

In `handleMouse`, the card height is no longer fixed at 6. We need to approximate based on expanded state:

```go
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	if len(m.nodes) == 0 {
		return m, nil
	}

	innerWidth := m.width - 2
	minCardWidth := 40
	numCols := innerWidth / minCardWidth
	if numCols < 1 {
		numCols = 1
	}
	if numCols > len(m.nodes) {
		numCols = len(m.nodes)
	}
	cardWidth := innerWidth / numCols

	// Estimate card height: base 6 (border + 3 content + border)
	// + process lines when expanded
	// Use max processes across nodes in the same row for row height
	gridStartY := 2

	clickX := msg.X - 1
	clickY := msg.Y - gridStartY

	if clickX < 0 || clickY < 0 {
		return m, nil
	}

	col := clickX / cardWidth
	if col >= numCols {
		col = numCols - 1
	}

	// Walk rows to find which row was clicked
	nodeIdx := -1
	currentY := 0
	for rowStart := 0; rowStart < len(m.nodes); rowStart += numCols {
		rowEnd := rowStart + numCols
		if rowEnd > len(m.nodes) {
			rowEnd = len(m.nodes)
		}

		// Find max card height in this row
		maxHeight := 6 // base: top border + 3 content + bottom border + gap
		if m.expanded {
			for i := rowStart; i < rowEnd; i++ {
				procCount := len(m.nodes[i].ActiveUsers())
				h := 6 + procCount
				if h > maxHeight {
					maxHeight = h
				}
			}
		}

		if clickY >= currentY && clickY < currentY+maxHeight {
			idx := rowStart + col
			if idx < rowEnd {
				nodeIdx = idx
			}
			break
		}
		currentY += maxHeight
	}

	if nodeIdx >= 0 && nodeIdx < len(m.nodes) {
		m.selectedIdx = nodeIdx
		node := m.nodes[nodeIdx]
		if node.IsOnline() {
			m.bottomPanel = PanelDetail
			m.detailNode = nil
			m.detailSys = nil
			return m, m.queryDetail(node.Hostname)
		}
	}

	return m, nil
}
```

- [ ] **Step 6: Update group switching to recompute display names**

In the `"g"` key handler, after updating `m.hosts`, recompute display names:

```go
	case "g":
		if len(m.groupNames) > 0 {
			m.currentGroup++
			if m.currentGroup >= len(m.groupNames) {
				m.currentGroup = -1
			}
			if m.currentGroup >= 0 {
				m.hosts = m.groups[m.groupNames[m.currentGroup]]
			} else {
				m.hosts = m.allHosts
			}
			m.displayNames = components.ComputeDisplayNames(m.hosts)
			m.selectedIdx = 0
		}
		return m, nil
```

- [ ] **Step 7: Build to verify compilation**

Run: `go build ./...`
Expected: success.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: inline process display in cards, remove bottom process panel"
```

---

### Task 7: Update cmd/root.go

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Remove --processes flag and simplify**

Remove the `flagProcesses` variable and its flag registration. Update `run()`:

```go
// Remove from var block:
// flagProcesses bool

// Remove from init():
// rootCmd.Flags().BoolVarP(&flagProcesses, "processes", "p", false, "Toggle processes off (default: on)")

// In run(), remove:
// showProcs := true
// if cmd.Flags().Changed("processes") {
//     showProcs = !flagProcesses
// }

// Update static mode call — pass showProcs as true always:
// renderStatic(results, cfg.Interval, termWidth, true)

// Update NewModel call — remove showProcs parameter:
// showProcesses: true is now hardcoded in NewModel
```

The `NewModel` signature still accepts `showProcesses bool` — we just always pass `true`. No need to change the signature itself since it's internal.

- [ ] **Step 2: Update renderStatic to use new palette (automatic — colors come from styles.go)**

No code changes needed — `renderStatic` already calls `components.RenderHeader`, `components.RenderGradientBar`, etc., which use the updated palette constants.

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "refactor: remove --processes flag, processes always shown inline"
```

---

### Task 8: Update Remaining Components for New Palette

**Files:**
- Modify: `internal/tui/components/proctable.go`
- Modify: `internal/tui/components/nodedetail.go`
- Modify: `internal/tui/components/help.go`

- [ ] **Step 1: Verify all three files compile with new palette**

Since color constant names are unchanged (`ColorFg`, `ColorDim`, `ColorBorder`, `ColorAccent`, `ColorGreen`, `ColorRed`, `ColorYellow`), these files automatically use the new palette without code changes.

Run: `go build ./...`
Expected: success.

- [ ] **Step 2: Remove duplicate formatGPURange from proctable.go**

The `formatGPURange` function exists in both `proctable.go` (line 122) and the new `nodecard.go`. Remove it from `proctable.go` since `nodecard.go` now exports it (both are in the same package, so it's accessible).

Actually — both are in `package components`, so having the same function name in two files causes a compile error. Delete the `formatGPURange` function from `proctable.go` entirely (lines 122-148).

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Run all existing tests**

Run: `go test ./... -v`
Expected: all PASS (existing tests + new hostname tests).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/components/proctable.go
git commit -m "refactor: deduplicate formatGPURange, palette auto-updated"
```

---

### Task 9: Build, Test on Live Cluster, Verify

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `go build -o node-monitor .`
Expected: success, produces binary.

- [ ] **Step 2: Test static mode**

Run: `./node-monitor --static`
Expected: output with new muted color palette, full-block bars `█`.

- [ ] **Step 3: Test TUI mode**

Run: `./node-monitor`
Expected:
- Cards show inline processes for nodes with GPU activity
- Idle nodes show `idle` in dim italic
- Hostnames are truncated (showing `115`, `116`, etc.)
- `p` key collapses/expands process lines
- Card borders align properly (╮ matches │)
- Enter on a card opens detail in bottom panel
- Colors are muted, not neon

- [ ] **Step 4: Verify mouse click works with variable card heights**

Click on different cards in the grid. Verify the correct node is selected even when cards in the same row have different heights.

- [ ] **Step 5: Final commit if any adjustments were needed**

```bash
git add -A
git commit -m "fix: adjustments from live testing"
```

# TUI Visual Polish — Technical Findings

## Finding 1: lipgloss `Width()` Excludes Border Characters

**Discovery context:** Card grid consistently overflowed the outer frame's right border.

**Analysis:**

In lipgloss v1.1.0, `Style.Width(n)` sets the width of the content area **including horizontal padding but excluding borders**. This means:

```
lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).  // adds │ on left + │ on right
    Padding(0, 1).                     // 1 char padding each side (inside Width)
    Width(w)                           // content + padding = w chars

Visual width = w + 2  (border chars added OUTSIDE)
```

**Impact on card grid:**

The original code computed `cardWidth = width/numCols - 1`, meaning each card's visual width was `width/numCols + 1`. For N columns, the total grid width was `width + N`, overflowing by N characters.

**Fix:** `cardWidth = width/numCols - 2` — subtracts the full border overhead.

**Verification:** With 6 nodes at terminal width 200:
- `gridWidth = 200 - 3 = 197`, `numCols = 4`
- `cardWidth = 197/4 - 2 = 47`, visual width = 49
- Total = 4 × 49 = 196 ≤ 197 ✓

**Related:** The `innerWidth = width - 4` in `renderCondensedCard` accounts for both border (2) and padding (2), which is correct for computing the text area within the card. This was independently correct.

---

## Finding 2: embedTitle Width Must Match Rendered Body, Not Style Parameter

**Discovery context:** Card top-right corner `╮` was consistently 1-2 chars offset from the right `│` border.

**Analysis:**

The `embedTitle` function replaces the lipgloss-generated top border line with a custom `╭─┤ title ├──╮` line. The original code passed the `width` style parameter as the target width. But the actual rendered body lines have a visual width of `width + 2` (due to borders being outside Width). The title line was therefore 2 chars shorter than the body.

**Fix:** Measure the actual body line width via `lipgloss.Width(lines[1])` and use that as the target:

```go
rendered := style.Render(content)
lines := strings.Split(rendered, "\n")
if len(lines) > 1 {
    bodyWidth := lipgloss.Width(lines[1])  // measure real width
    lines[0] = embedTitle(lines[0], titleText, borderColor, bodyWidth)
}
```

This is self-adapting — works regardless of lipgloss version's Width semantics.

---

## Finding 3: QueryNode Must Fetch Process Data for Inline Display

**Discovery context:** Cards showed GPU utilization/memory bars but no user/process information.

**Analysis:**

The architecture has two query paths:

| Function | Command | Data Retrieved |
|----------|---------|---------------|
| `QueryNode` (tick) | `ListViewCommand` | GPU index, util, mem, name — **no processes** |
| `QueryNodeDetail` (on click) | `DetailViewCommand` | GPU + processes + UUID map + users + system info |

The card rendering code (`renderCardProcesses`) reads `gpu.Processes`, which was always empty because `QueryNode` → `parseGPUOutput` only parsed the basic CSV fields.

**Fix:** Created `ListWithProcessesCommand()` — same as `DetailViewCommand` but without the `---SYSTEM---` section (lighter weight). Changed `QueryNode` to use this command and `parseDetailOutput` parser. The parser gracefully handles a missing system section (returns `nil`).

**Performance note:** The extra SSH commands per tick are: `nvidia-smi --query-compute-apps`, `nvidia-smi --query-gpu` (UUID map), and `ps -eo pid,user`. These are fast local commands on each node. For 6 nodes at 2s interval, overhead is negligible.

---

## Finding 4: Detail Panel Requires Active Refresh

**Discovery context:** When clicking a node to open the detail panel, the per-GPU bars showed initial values but never updated. The top card grid updated normally.

**Analysis:**

The tick cycle dispatches `queryNodes()` which updates `m.nodes` via `nodesUpdatedMsg`. The detail panel data (`m.detailNode`, `m.detailSys`) was only fetched once via `queryDetail()` when the user clicked a node. Subsequent ticks did not re-query the detail.

**Fix:** In the `nodesUpdatedMsg` handler, if the detail panel is open, dispatch a `queryDetail` command:

```go
case nodesUpdatedMsg:
    m.nodes = sortNodes(...)
    // ... bounds check ...
    if m.bottomPanel == PanelDetail && m.detailNode != nil {
        return m, m.queryDetail(m.detailNode.Hostname)
    }
    return m, nil
```

This means the detail refreshes at the same rate as the main grid (every tick interval).

---

## Finding 5: Terminal Resize Requires Fixed-Size Output

**Discovery context:** Dragging the terminal window caused layout corruption — ghost content, overlapping lines, partial renders.

**Analysis:**

Bubble Tea uses alternate screen mode and redraws on each `View()` call. When the terminal is resized, a `WindowSizeMsg` updates `m.width` and `m.height`, triggering a redraw. The problem was that `RenderOuterFrame` produced a variable number of output lines:

- If content was shorter than terminal: bottom of screen showed stale content from the previous render
- If content was taller than terminal: output scrolled past the visible area, causing visual artifacts

btop avoids this by always producing exactly `height` lines of output, each exactly `width` chars wide.

**Fix:** `RenderOuterFrame` now:

1. Calculates `availableLines = height - 2` (top + bottom border)
2. Truncates body if `len(bodyLines) > availableLines`
3. Pads with empty bordered lines (`│` + spaces + `│`) if fewer lines
4. Pre-allocates `framedLines` with `make([]string, 0, height)` for efficiency

**Result:** Output is always exactly `height` lines. No ghost content, no scrolling artifacts.

---

## Finding 6: Pure Black Background via ANSI Reset Replacement

**Discovery context:** btop has a distinctive pure black background that enhances contrast with its muted color palette. node-monitor used the terminal's default background.

**Analysis — approaches considered:**

| Approach | Pros | Cons |
|----------|------|------|
| `Background(ColorBg)` on every lipgloss style | Complete coverage | Very verbose; affects static mode too; lipgloss `\033[0m` resets break it mid-line |
| OSC 11 (`\033]11;#000000\033\\`) | Changes terminal bg globally | Persists after exit; complex to save/restore original |
| `lipgloss.Place` with `WithWhitespaceBackground` | Clean API | Only affects Place-added whitespace, not content spaces |
| **ANSI reset replacement** | Simple, reliable, covers everything | Slight string processing overhead per frame |

**Chosen approach — ANSI reset replacement:**

```go
func ApplyBackground(s string) string {
    bgRestore := "\x1b[0;48;2;0;0;0m"
    s = strings.ReplaceAll(s, "\x1b[0m", bgRestore)
    return "\x1b[48;2;0;0;0m" + s + "\x1b[0m"
}
```

**How it works:**

1. lipgloss/termenv uses `\x1b[0m` (SGR reset) to end styled spans
2. A normal reset clears ALL attributes including background → subsequent spaces use terminal default bg
3. Our replacement `\x1b[0;48;2;0;0;0m` resets then immediately re-sets the background to RGB(0,0,0)
4. The initial `\x1b[48;2;0;0;0m` sets background for the first characters before any reset
5. The final `\x1b[0m` cleanly resets for Bubble Tea's next frame

**Safety analysis:**
- `\x1b[0m` and `\x1b[0;48;2;0;0;0m` are different strings — no recursive replacement
- termenv consistently uses `\x1b[0m` for reset (not `\x1b[m` shorthand)
- Performance: ~50KB string per frame, `ReplaceAll` is O(n), negligible at 2s interval
- Only applied in TUI mode (`View()`), not static mode

---

## Finding 7: Grid Width Must Account for Body Margin

**Discovery context:** Rightmost card overflow was partially caused by a left-margin space not subtracted from the grid width budget.

**Analysis:**

In `View()`:
```go
innerWidth := m.width - 2          // available between outer frame │ borders
nodeGrid := RenderNodeGrid(..., innerWidth, ...)  // grid rendered to fill innerWidth

for _, line := range strings.Split(nodeGrid, "\n") {
    bodyLines = append(bodyLines, " "+line)  // adds 1 char left margin
}
```

The grid used the full `innerWidth`, but then each line got a `" "` prefix. Total body line width: `1 + gridLineWidth`. This exceeded `innerWidth` (the space between frame borders), causing overflow into or past the right `│`.

**Fix:** Pass `innerWidth - 1` to `RenderNodeGrid`:
```go
gridWidth := innerWidth - 1  // account for left margin
nodeGrid := RenderNodeGrid(..., gridWidth, ...)
```

The mouse handler was also updated to use matching calculations: `gridWidth = m.width - 3` and `clickX = msg.X - 2` (frame border + margin).

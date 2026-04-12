# TUI Visual Polish & Process Display Design Spec

## Goal

Refine the existing Go TUI to match btop++ aesthetic quality: lower-saturation color palette, full-block bar characters, inline process display in node cards, smart hostname truncation, and fix card border alignment.

## Changes Overview

### 1. Color Palette: Dracula → btop++ Muted

Replace the high-saturation Dracula palette with calmer, layered tones. Bright colors reserved for data-carrying elements only.

| Purpose | Old (Dracula) | New (btop-inspired) |
|---------|--------------|-------------------|
| Primary text | `#f8f8f2` | `#c9d1d9` |
| Dim/labels | `#6272a4` | `#484f58` |
| Border/frame | `#44475a` | `#30363d` |
| Meter background | `#44475a` | `#21262d` |
| Accent (keys, usernames) | `#8be9fd` (cyan) | `#388bfd` (blue) |
| Green (low util) | `#50fa7b` | `#2ea043` |
| Yellow/amber (mid util) | `#f1fa8c` | `#d29922` |
| Orange | `#ffb86c` | `#cf6e2a` |
| Red (high util) | `#ff5555` | `#da3633` |
| Selection highlight | `#f8f8f2` | `#e6edf3` |

Gradient definitions updated accordingly:
- **Utilization**: `#2ea043` → `#d29922` → `#da3633`
- **Memory**: `#388bfd` → `#d29922` → `#da3633`
- **GPU heatmap**: same as utilization gradient

### 2. Bar Character Change

Replace `■` (U+25A0) filled blocks with `█` (U+2588) full blocks for denser, smoother visual fill. Unfilled `░` (U+2591) stays but uses new meter background `#21262d`.

### 3. Processes Inline in Node Cards (Option B: Seamless)

Default behavior: each node card shows its GPU processes directly below the GPU heatmap line, with no divider.

**Expanded card layout (default):**
```
╭─┤ 🔥 115 ├──────────────────────╮
│ Util █████████████████████ 100% │
│ Mem  ██████████████████░░░ 535G │
│ GPU  ████████ 8×H100           │
│ alice 0-7 535G train.py        │
╰──────────────────────────────────╯
```

**Multi-user card:**
```
╭─┤ ✓ 116 ├───────────────────────╮
│ Util ░░░░░░░░░░░░░░░░░░░░   0% │
│ Mem  ██████░░░░░░░░░░░░░░ 200G │
│ GPU  ████████ 8×H100           │
│ bob   0,3 160G torchrun pr..   │
│ carol 5    40G eval.py         │
╰──────────────────────────────────╯
```

**Idle node (no processes):**
```
╭─┤ ✓ 120 ├───────────────────────╮
│ Util ░░░░░░░░░░░░░░░░░░░░   0% │
│ Mem  ░░░░░░░░░░░░░░░░░░░░   8M │
│ GPU  ████████ 8×H100  idle     │
╰──────────────────────────────────╯
```

Idle nodes show `idle` in dim italic on the GPU line, keeping the card at minimum 4-line height (no extra process lines).

**Process line format:** `username gpuRange memUsed cmdShort`
- Username: blue (`#388bfd`), left-aligned, max 8 chars
- GPU range: green (`#2ea043`), compact format (`0-7`, `0,3`, `5`)
- Memory: primary text color, right-aligned within 5 chars
- Command: dim (`#484f58`), truncated with `..` to fit card width

**Process aggregation:** Same as current — aggregate by user per node. Show GPU indices used, total memory, and first command name (basename only, strip path).

### 4. Collapse/Expand Interaction

- `p` key: toggle ALL cards between expanded (with processes) and collapsed (4-line: Util/Mem/GPU only)
- Default state: expanded
- `Enter` or click on a card: open full detail view in bottom panel (per-GPU bars, full process list, system info)
- Card height becomes variable: 4 lines base + N process user lines when expanded

**Bottom panel changes:** With processes now inline in cards, the global process table in the bottom panel is removed. The bottom panel is used exclusively for the node detail view (activated by Enter/click). When no detail is open, the bottom panel is hidden and cards fill the full height.

### 5. Smart Hostname Truncation

Zero-config, automatic approach with two layers:

**Layer 1 — Common prefix stripping:**
When there are 2+ nodes, find the longest common prefix of all hostnames. If the prefix ends at a natural boundary (`.`, `-`, or digit-to-non-digit transition), strip it. Display only the distinguishing suffix.

Example: `["host-10-240-99-115", "host-10-240-99-116", ..., "host-10-240-99-120"]`
- Common prefix: `host-10-240-99-`
- Display names: `115`, `116`, `117`, `118`, `119`, `120`

Edge case: if only 1 node, show full hostname. If stripping would leave empty string, show full hostname.

**Layer 2 — Tail truncation fallback:**
If display name still exceeds `cardWidth - 16` chars (leaving room for icon + border), truncate from the left with `..` prefix: `..some-long-suffix`.

### 6. Card Border Alignment Fix

The current `embedTitle` function in `nodecard.go` has a width calculation bug causing the top-right `╮` corner to misalign with the right `│` border. Root cause: `lipgloss.Width()` may not correctly account for emoji/unicode widths in the title string, and the card `Width` set via lipgloss style includes padding.

Fix approach:
- Calculate visible character width of title string manually, accounting for emoji as width 2
- Ensure `embedTitle` output width exactly matches the lipgloss-rendered card body width
- Verify alignment by testing with varying hostname lengths

### 7. Outer Frame Divider Connection

When the bottom panel is shown, the divider line should properly connect to the outer frame side borders using `├` and `┤` instead of floating `─` segments.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/components/styles.go` | Update | New color palette constants, updated gradient RGB values |
| `internal/tui/components/gpubar.go` | Update | Change `■` to `█` in `RenderGradientBar` |
| `internal/tui/components/nodecard.go` | Rewrite | Add process lines, variable card height, fix `embedTitle` alignment, smart hostname |
| `internal/tui/components/header.go` | Update | New palette colors, fix outer frame divider connection |
| `internal/tui/components/proctable.go` | Update | New palette colors |
| `internal/tui/components/nodedetail.go` | Update | New palette colors |
| `internal/tui/components/help.go` | Update | New palette colors |
| `internal/tui/app.go` | Update | Default `showProcesses=true` means cards expanded, `p` toggles card expansion, adjust mouse click regions for variable card heights |
| `cmd/root.go` | Update | Remove `--processes` flag (processes always inline, `p` key toggles at runtime) |

## Not Changed

- Overall layout architecture (outer frame, 3-zone vertical)
- Keyboard shortcuts (same keys, same behavior)
- SSH query logic, data model, config system
- Detail view content (just palette update)
- Static mode (palette update only)

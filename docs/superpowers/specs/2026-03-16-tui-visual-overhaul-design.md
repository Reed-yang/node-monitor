# TUI Visual Overhaul Design Spec

## Goal

Transform the current Go TUI from a basic functional display into a polished, btop++-inspired interactive dashboard with mouse support. Remove the panel/compact view distinction — deliver one unified, information-dense, visually excellent default view.

## Reference Systems

- **btop++**: Gradient progress bars (per-character coloring), box-drawing borders with embedded titles (`╭─┤ Title ├──╮`), three-color gradient interpolation, meter_bg for unfilled portions, information density via color-as-data
- **lazygit**: Active/inactive panel focus via border color, context-sensitive keybinding hints, clean selection highlighting, mouse-clickable UI elements

## Layout Architecture

Three-zone vertical layout wrapped in a single outer frame:

```
╭─┤ GPU Cluster Monitor ├── 4 nodes │ 32 GPUs │ ⚡73% │ 💾285G ── s:sort /:search ?:help ─╮
├──────────────────────────────────────────────────────────────────────────────────────────┤
│  ╭─┤ visko-1 ⚡ ├──────────╮  ╭─┤ visko-2 🔥 ├──────────╮                              │
│  │ Util ■■■■■■■■■■■░░░ 67% │  │ Util ■■■■■■■■■■■■■░ 91% │                              │
│  │ Mem  ▰▰▰▰▰▰▰░░░░░ 285G │  │ Mem  ▰▰▰▰▰▰▰▰▰▰▰▰ 595G │    ← Node Grid              │
│  │ GPU ██▓▒██████ 8×A100   │  │ GPU ████████ 8×A100      │                              │
│  ╰──────────────────────────╯  ╰──────────────────────────╯                              │
│  ╭─┤ visko-3 ✓ ├──────────╮  ╭─┤ visko-4 ✗ ├──────────╮                                │
│  │ Util ■■░░░░░░░░░░░  12% │  │ ⚠ SSH timed out        │                                │
│  │ Mem  ▰▰░░░░░░░░░░░  37G │  │                         │                                │
│  │ GPU ▒▒▒▒▒▒▒▒ 8×A100     │  │                         │                                │
│  ╰──────────────────────────╯  ╰──────────────────────────╯                              │
├─┤ Processes ├────────────────────────────────────────────────────────────────────────────┤
│  USER     NODE      GPU    MEM     CMD                                                   │
│  alice    visko-1   0,1   68.0G    python train.py                                       │
│  bob      visko-2   0-7  595.0G    torchrun pretrain.py                                  │
╰──────────────────────────────────────────────────────────────────────────────────────────╯
```

### Zone 1: Header (embedded in outer frame top border)

Single line embedded in the outer `╭──╮` border using btop's `┤ Title ├` pattern:
- Left: `┤ GPU Cluster Monitor ├`
- Center: stats — node count, GPU count, avg utilization (colored), total memory (colored)
- Right: clickable keybinding hints — `s:sort  /:search  ?:help`

Each keybinding hint is a clickable region. The key letter is rendered in a bright accent color (cyan), the description in dim text.

### Zone 2: Node Grid

Auto-layout grid of condensed node cards. Column count = `floor(termWidth / minCardWidth)`, where `minCardWidth = 40`. Cards expand equally to fill available width.

#### Condensed Node Card (4 lines of content)

```
╭─┤ hostname icon ├──────────────╮
│ Util ■■■■■■■■■■■■░░░░░░░  67% │
│ Mem  ■■■■■■■■░░░░░░░░░░░ 285G │
│ GPU ██▓▒██████ 8×A100 alice..  │
╰────────────────────────────────╯
```

**Line 1 — Aggregate Utilization Bar:**
- Label `Util` in dim text
- Gradient progress bar: 20 chars wide, each `■` independently colored via 3-color interpolation (green `#50fa7b` → yellow `#f1fa8c` → red `#ff5555`)
- Unfilled portion: `░` in dark gray (`#44475a`), NOT bright — reduces visual noise
- Percentage right-aligned, colored same as the bar's dominant color, bold

**Line 2 — Aggregate Memory Bar:**
- Label `Mem ` in dim text
- Gradient bar with memory color scale (cyan `#8be9fd` → yellow `#f1fa8c` → red `#ff5555`)
- Memory value: `UsedG/TotalG` or just `UsedG` if space is tight, right-aligned

**Line 3 — GPU Heatmap Strip + Metadata:**
- Label `GPU` in dim text
- Heatmap: one `█` character per GPU, each colored by that GPU's individual utilization (same green→red scale). This gives instant per-GPU visibility in minimal space.
- GPU model name (extracted from nvidia-smi if available, or `N×GPU` count)
- Active usernames (comma-separated, truncated to fit)

#### Card Border Styling

| State | Border Color | Title Color |
|-------|-------------|-------------|
| Online, selected (keyboard/mouse) | Bright white `#f8f8f2` | White bold |
| Online, avg util > 80% | Red `#ff5555` | Red + icon 🔥 |
| Online, avg util > 50% | Yellow `#f1fa8c` | Yellow + icon ⚡ |
| Online, avg util ≤ 50% | Green `#50fa7b` | Green + icon ✓ |
| Offline | Dark red `#ff5555` dim | Red + icon ✗ |

Selected card border overrides the utilization color with bright white, making focus unambiguous.

#### Offline Node Card

```
╭─┤ visko-4 ✗ ├──────────────────╮
│ ⚠ SSH connection timed out     │
│                                 │
│                                 │
╰─────────────────────────────────╯
```

Entire card rendered in dim red/gray. Error message truncated to card width.

### Zone 3: Bottom Panel

Occupies the lower portion of the outer frame. Two modes:

**Default — Global Processes Table:**
- Title embedded in divider: `├─┤ Processes ├──`
- Columns: USER, NODE, GPU, MEM, CMD
- Rows sorted by memory usage descending
- Aggregated by user+node (not per-PID)
- If no processes, shows `No GPU processes running` in dim text
- Scrollable with mouse wheel or j/k when this panel is focused

**After clicking/selecting a node — Node Detail View:**
- Title changes to: `├─┤ visko-2 Detail ├──`
- Top section: per-GPU bars in 2-column layout (GPU pairs side by side)
- Middle section: process table filtered to this node
- Bottom section: system info (Load, RAM, Driver) in a single line
- Esc or clicking another node exits detail view

## Gradient Progress Bar System

### 3-Color Interpolation

Each gradient is defined by 3 colors (start, mid, end). A 101-element color array is pre-computed:
- Index 0-50: interpolate start → mid
- Index 51-100: interpolate mid → end

When rendering a bar of width W at value V%:
- For each position `i` (0 to W-1):
  - Threshold = `i * 100 / W`
  - If threshold ≤ V: render `■` colored by `gradient[threshold]`
  - Else: render `░` colored by meter_bg (`#44475a`)

This produces a smooth left-to-right color transition within a single bar.

### Gradient Definitions

| Metric | Start | Mid | End |
|--------|-------|-----|-----|
| Utilization | `#50fa7b` (green) | `#f1fa8c` (yellow) | `#ff5555` (red) |
| Memory | `#8be9fd` (cyan) | `#f1fa8c` (yellow) | `#ff5555` (red) |
| GPU Heatmap | Same as Utilization gradient | | |

### Bar Characters

- Filled: `■` (U+25A0) — matches btop exactly
- Unfilled: `░` (U+2591) in `#44475a` — dark, low visual noise
- Heatmap: `█` (U+2588) per GPU — one char = one GPU

## Mouse Interaction

Enable via `tea.WithMouseCellMotion()` in Bubble Tea.

### Click Regions

The app tracks clickable regions by recording bounding boxes `(x1, y1, x2, y2)` during each `View()` render pass. On `tea.MouseMsg`, the click position is tested against recorded regions.

| Region | Click Action |
|--------|-------------|
| Node card area | Select that node (highlight border) |
| Node card double-click or Enter | Open node detail in bottom panel |
| Process table row | Highlight corresponding GPU in the node grid |
| Header keybinding hint (`s:sort`) | Trigger sort cycle |
| Header keybinding hint (`/:search`) | Enter search mode |
| Header keybinding hint (`?:help`) | Toggle help overlay |
| Bottom panel (scroll) | Mouse wheel scrolls content |

### Click Region Tracking

```go
type ClickRegion struct {
    X1, Y1, X2, Y2 int
    Action          string   // "select-node", "detail-node", "sort", "search", "help"
    Payload         string   // e.g., hostname for node actions
}
```

Regions are rebuilt every render cycle (they change on resize/scroll). Stored as `[]ClickRegion` on the Model.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate node selection |
| `Enter` | Open selected node detail in bottom panel |
| `Esc` | Close detail view / Close search / Exit |
| `q` | Quit |
| `p` | Toggle process panel visibility |
| `s` | Cycle sort: name → utilization → memory |
| `g` | Cycle node group filter |
| `/` | Enter search mode (filter nodes by name) |
| `?` | Toggle help overlay |
| `Tab` | Toggle focus between node grid and bottom panel |

## Removed Features

- **`--compact` / `-c` flag**: Removed. No view mode toggle. One unified view.
- **`Tab` view toggle**: Repurposed to toggle focus between zones.
- **Panel view**: Removed entirely. The condensed card grid IS the only view.

## Modified CLI Flags

```
node-monitor [flags]

Flags:
  -n, --nodes string     Comma-separated node list
  -g, --group string     Node group from config
  -i, --interval float   Refresh interval in seconds (default: 2.0)
  -w, --workers int      Max parallel SSH connections (default: 8)
  -s, --static           Print once and exit (no TUI)
  -p, --processes        Show processes panel on start (default: true)
  -d, --debug            Verbose SSH error output
  -v, --version          Show version
  -h, --help             Show help
```

`--compact` removed. `--processes` default changed to `true` (panel always shown, `p` toggles).

## Static Mode (`-s`)

Static mode renders the same visual layout but without the outer frame and without mouse/keyboard interactivity. Output goes to stdout and the process exits. This is for scripting and piping.

Static output:
```
GPU Cluster Monitor │ 4 nodes │ 32 GPUs │ ⚡73% │ 💾285G │ 2026-03-16 15:04:05

 visko-1 ⚡ 67%   Util ■■■■■■■■■■■■■░░░░░░░  Mem 285/637G  GPU ██▓▒██████
 visko-2 🔥 91%   Util ■■■■■■■■■■■■■■■■■■░░  Mem 595/637G  GPU ████████
 visko-3 ✓  12%   Util ■■░░░░░░░░░░░░░░░░░░  Mem  37/637G  GPU ▒▒▒▒▒▒▒▒
 visko-4 ✗         ⚠ SSH connection timed out

Processes:
 USER     NODE      GPU    MEM     CMD
 alice    visko-1   0,1   68.0G    python train.py
 bob      visko-2   0-7  595.0G    torchrun pretrain.py
```

## Color Palette (Dracula-inspired)

| Purpose | Color | Hex |
|---------|-------|-----|
| Background | Terminal default | — |
| Primary text | Light gray | `#f8f8f2` |
| Dim/secondary text | Gray | `#6272a4` |
| Outer frame border | Subtle gray | `#44475a` |
| Active panel border | Bright white | `#f8f8f2` |
| Accent (keybindings) | Cyan | `#8be9fd` |
| Success/low | Green | `#50fa7b` |
| Warning/medium | Yellow | `#f1fa8c` |
| Danger/high | Red | `#ff5555` |
| Danger/orange | Orange | `#ffb86c` |
| Meter background | Dark gray | `#44475a` |
| Offline/error | Red dim | `#ff5555` at 50% opacity (faint) |

## GPU Model Detection

The nvidia-smi query for the list view is extended to include GPU name:

```bash
nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total,name --format=csv,noheader,nounits
```

Output: `0, 87, 74900, 79600, NVIDIA A100-SXM4-80GB`

The GPU name is parsed, shortened to a display name (e.g., `A100`), and shown in the heatmap line. All GPUs on a node are assumed same model — use the first GPU's name with count prefix: `8×A100`.

This requires updating `model.GPUInfo` to include a `Name string` field, and updating the parse logic.

## Data Model Changes

```go
// GPUInfo — add Name field
type GPUInfo struct {
    Index       int
    Utilization int
    MemoryUsed  int
    MemoryTotal int
    Name        string       // NEW: GPU model name
    Processes   []GPUProcess
}
```

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/components/styles.go` | Rewrite | Dracula palette, gradient system, border styles |
| `internal/tui/components/gpubar.go` | Rewrite | Gradient bars, GPU heatmap strip |
| `internal/tui/components/nodelist.go` | Rewrite → `nodecard.go` | Condensed node cards with grid layout |
| `internal/tui/components/header.go` | Rewrite | Embedded in outer frame, clickable hints |
| `internal/tui/components/nodedetail.go` | Rewrite | Inline detail in bottom panel |
| `internal/tui/components/help.go` | Minor update | Adjust shortcuts list |
| `internal/tui/components/frame.go` | New | Outer frame rendering with embedded header |
| `internal/tui/components/proctable.go` | New | Process table for bottom panel |
| `internal/tui/app.go` | Rewrite | Mouse support, click regions, unified view, focus management |
| `internal/model/types.go` | Update | Add `Name` to GPUInfo |
| `internal/ssh/parse.go` | Update | Parse GPU name from nvidia-smi |
| `internal/ssh/parse_test.go` | Update | Tests for new CSV column |
| `cmd/root.go` | Update | Remove `--compact`, change `--processes` default, add mouse |

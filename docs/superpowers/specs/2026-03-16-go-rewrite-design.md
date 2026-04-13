# Node Monitor: Go Rewrite Design Spec

## Overview

Rewrite the Python-based `node-monitor` CLI tool in Go to solve three core pain points:
1. **Distribution** — eliminate Python environment dependency, ship a single static binary
2. **UI/UX** — upgrade from Rich static output to an interactive TUI with Bubble Tea
3. **Developer experience** — leverage Go's goroutines for cleaner concurrent SSH, native SSH library for connection reuse

The tool monitors GPU resources across Slurm cluster nodes via SSH, displaying real-time utilization, memory, and process information.

## Target Users

- Primary: the author, across multiple machines/environments
- Secondary: team members who may not manage Python environments
- Tertiary: open-source users who want a one-command install

## Architecture

### Project Structure

```
node-monitor-go/
├── main.go                    # Entry point, version embedding
├── cmd/
│   └── root.go                # Cobra CLI definition
├── internal/
│   ├── config/
│   │   └── config.go          # Viper-based config loading & merging
│   ├── ssh/
│   │   └── query.go           # SSH connection pool + nvidia-smi queries
│   ├── slurm/
│   │   └── detect.go          # sinfo parsing, auto-discover nodes
│   ├── model/
│   │   └── types.go           # GPUInfo, NodeStatus, GPUProcess data types
│   └── tui/
│       ├── app.go             # Bubble Tea main program (Model/Update/View)
│       ├── components/
│       │   ├── header.go      # Top status bar
│       │   ├── nodelist.go    # Node list (selectable, sortable)
│       │   ├── nodedetail.go  # Node detail drill-down view
│       │   ├── gpubar.go      # GPU utilization/memory progress bar
│       │   └── help.go        # Keyboard shortcut help overlay
│       └── styles.go          # Lip Gloss style definitions
├── configs/
│   └── default.toml           # Example default configuration
├── go.mod
├── go.sum
├── Makefile
└── .goreleaser.yaml           # Multi-platform automated release
```

### Core Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | Styling and layout |
| `github.com/charmbracelet/bubbles` | Pre-built components (spinner, table, viewport) |
| `github.com/spf13/cobra` | CLI argument parsing |
| `github.com/spf13/viper` | Config file management |
| `golang.org/x/crypto/ssh` | Native SSH client |
| `github.com/kevinburke/ssh_config` | Parse `~/.ssh/config` |

### Data Flow

```
CLI args + config.toml → Config
        ↓
Slurm detect / manual nodes → []string
        ↓
SSH goroutines (parallel) → []NodeStatus
        ↓
Bubble Tea Model.Update() → state update
        ↓
Bubble Tea Model.View() → Lip Gloss render → terminal
        ↓
Tick (interval) → back to SSH query
```

## Configuration System

### Priority (high → low)

1. CLI flags (highest)
2. `~/.config/node-monitor/config.toml`
3. Built-in defaults

### Config File Format

```toml
# Default node list (omit to auto-detect from Slurm)
nodes = ["visko-1", "visko-2", "visko-3"]

# Refresh interval (seconds)
interval = 2.0

# Max parallel SSH connections
workers = 8

# Default view mode: "panel" | "compact"
view = "compact"

# Show process info
processes = false

# SSH configuration
[ssh]
connect_timeout = 5  # SSH connection timeout (seconds)
command_timeout = 10 # Command execution timeout (seconds)
identity_file = ""   # Private key path (empty = default)
user = ""            # SSH username (empty = current user)

# Node groups (optional, for large cluster management)
[groups]
train = ["visko-1", "visko-2", "visko-3", "visko-4"]
inference = ["infer-1", "infer-2"]
```

Node groups allow `node-monitor --group train` to filter by category without typing full node lists.

## Interactive TUI Design

### Dual Mode

| Mode | Trigger | Purpose |
|------|---------|---------|
| **Dashboard** (default) | `node-monitor` | Interactive TUI with keyboard navigation (alternate screen buffer) |
| **Static** | `node-monitor --static` / `-s` | Print once and exit, for scripts/pipes |

Note: The Python version's `--no-fullscreen` / `-F` flag is superseded by `--static`. In Dashboard mode, Bubble Tea always uses the alternate screen buffer. For non-interactive output (scrollable, pipeable), use `--static`.

### View Modes

Two view modes, togglable with `Tab` or settable via `--compact` / `-c` flag / config:

**Panel view** (default): Each node rendered as a bordered panel with GPU progress bars inside. Panels auto-arrange in columns based on terminal width (2+ columns at >= 130 chars). Each panel shows:
- Node hostname with status icon in panel title
- Per-GPU row: index, utilization bar, memory bar
- Summary footer: total GPUs, average utilization, total memory

**Compact view**: Dense table layout. When terminal width >= 120 chars, uses dual-column GPU layout (two GPUs per row) to maximize density. Each row shows: node name, GPU index, utilization %, utilization bar, memory used/total. Nodes separated by horizontal rules.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate node list |
| `Enter` | Enter node detail view |
| `Esc` / `q` | Back to list / exit |
| `Tab` | Toggle panel / compact view |
| `p` | Toggle process display |
| `s` | Cycle sort (name / utilization / memory) |
| `g` | Cycle node group filter |
| `/` | Search/filter nodes by name |
| `?` | Show keyboard shortcut help |

### Node Detail Drill-Down View

Activated by pressing Enter on a selected node. Displays:

- Per-GPU progress bars with utilization % and memory
- Process table: user, GPU indices, memory, command
- System info: CPU usage, system memory, uptime, driver version

Extra information is fetched only when entering detail view (not during list polling).

### Visual Style

- Lip Gloss gradient progress bars (green → yellow → red)
- Rounded borders with appropriate padding
- Selected row highlight (inverse or background color)
- Offline nodes in grey with visual distinction
- Balanced whitespace — compact but not cramped

### Color Coding

- **Utilization**: green (<30%) → yellow (<60%) → orange (<85%) → red (85%+)
- **Memory**: cyan (<50%) → yellow (<75%) → orange (<90%) → red (90%+)
- **Node status icons**: online (green) / medium load (yellow) / high load (red) / offline (grey)

## SSH & Concurrency

### Connection Pool

```
SSHPool {
    connections: map[hostname]*ssh.Client  (persistent cache)

    Get(host) → cached & alive? reuse : create new
    Query(host) → Get → session.Run(cmd)
    Close() → close all connections
}
```

Key behaviors:
- First query establishes connection, subsequent ticks reuse it
- Broken connections detected on error and automatically rebuilt (see Stale Connection Detection)
- Reads `~/.ssh/config` via `github.com/kevinburke/ssh_config` (ProxyJump, IdentityFile, etc.)
- Equivalent of SSH `BatchMode=yes`: no password auth callbacks registered, only publickey/agent auth
- Host key checking disabled by default (`ssh.InsecureIgnoreHostKey()`) with option to enable

### Parallel Query

Each tick launches one goroutine per node, all running concurrently:
- Global timeout via `context.WithTimeout`
- Single node failure does not block others
- Results collected via channel

### Stale Connection Detection

When `session.Run()` returns an SSH transport error, the pool discards the cached `*ssh.Client` and establishes a new connection on the next query. No keepalive; detection is error-driven (try-and-rebuild).

### Remote Commands

**List view query:**
```bash
nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits
```

**Detail view query** (all commands batched in a single SSH session, delimited by sentinel markers):
```bash
nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits
echo '---PROCESSES---'
nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory,process_name --format=csv,noheader,nounits
echo '---GPU_UUID_MAP---'
nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits
echo '---USERS---'
ps -eo pid,user --no-headers
echo '---SYSTEM---'
cat /proc/loadavg
free -b | head -2
cat /proc/driver/nvidia/version 2>/dev/null || echo 'N/A'
```

**Process resolution pipeline:**
1. Query compute apps → get GPU UUID, PID, memory, command
2. Query GPU UUID-to-index map → translate UUIDs to GPU indices
3. Query `ps` → resolve PIDs to usernames
4. Join results into `GPUProcess` structs

**System info:**
- CPU load: from `/proc/loadavg` (1/5/15 min load averages, not percentage)
- Memory: from `free -b` (total, used, available)
- Driver version: from `/proc/driver/nvidia/version`

## Distribution & Installation

### Build

GoReleaser with GitHub Actions. Git tag push triggers automated build for `linux/amd64` and `linux/arm64`.

Binary flags: `-s -w` (strip debug info) + embedded version via `-X main.version`.

### Installation Methods

| Method | Command | Audience |
|--------|---------|----------|
| GitHub Release | `curl -Lo node-monitor .../releases/latest/... && chmod +x && mv ~/.local/bin/` | Author, teammates |
| Install script | `curl -sSfL .../install.sh \| sh` | Friendly wrapper |
| Go install | `go install github.com/.../node-monitor@latest` | Go users |

### Comparison with Current Python Distribution

| Aspect | Python (current) | Go (new) |
|--------|------------------|----------|
| Install steps | clone → venv → pip install | download binary → add to PATH |
| Runtime deps | Python 3.8+, rich, click | None |
| Binary size | N/A (PyInstaller ~50MB) | ~10-12MB |
| Startup time | ~0.5-1s | ~10ms |
| Update | git pull → pip install | Download new binary |

## CLI Interface

```
node-monitor [flags]

Flags:
  -n, --nodes string     Comma-separated node list (default: auto-detect from Slurm)
  -g, --group string     Node group from config file
  -i, --interval float   Refresh interval in seconds (default: 2.0)
  -w, --workers int      Max parallel SSH connections (default: 8)
  -c, --compact          Start in compact view mode
  -s, --static           Print once and exit (no TUI)
  -p, --processes        Show GPU processes
  -d, --debug            Verbose SSH error output
  -v, --version          Show version
  -h, --help             Show help
```

## Non-Goals

- No web UI — this is a terminal tool
- No historical data storage — real-time only
- No alerting/notifications — out of scope for v1
- No Windows/macOS target — Linux-first (macOS possible later via Homebrew)

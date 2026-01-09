# Node Monitor 🖥️

A beautiful CLI tool for monitoring GPU resources across Slurm cluster nodes in real-time.

![Python](https://img.shields.io/badge/python-3.8+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## ✨ Features

- 🔍 **Auto-detect Slurm nodes** - Automatically discovers all nodes in your cluster via `sinfo`
- 🎮 **Real-time GPU monitoring** - Shows utilization and memory usage for each GPU
- 🎨 **Beautiful CLI** - Colorful, live-updating dashboard using Rich library
- ⚡ **Fast parallel queries** - Concurrent SSH connections for quick updates
- 📊 **Multi-column layout** - Auto-adapts to terminal width for optimal display
- 👤 **Process tracking** - Show GPU processes grouped by user (optional)

## 📦 Installation

```bash
# Clone the repository
git clone <repo-url>
cd node-monitor

# Install with pip
pip install -e .
```

## 🚀 Usage

### Basic Usage

```bash
# Auto-detect Slurm nodes and start monitoring
node-monitor
```

### Specify Nodes Manually

```bash
# Monitor specific nodes
node-monitor --nodes visko-1,visko-2,visko-3
# or short form
node-monitor -n visko-1,visko-2,visko-3
```

### Compact Table View

```bash
# Use compact table view (recommended for many nodes)
node-monitor -c
```

### Show GPU Processes

```bash
# Show GPU processes with user info
node-monitor -c -p
# or simply (auto-enables compact mode)
node-monitor -p
```

### Custom Refresh Interval

```bash
# Refresh every 5 seconds
node-monitor -i 5
```

### Scrollable Mode

```bash
# Disable fullscreen for terminal scrolling (useful for very many nodes)
node-monitor -c -F
```

### All Options

```
Usage: node-monitor [OPTIONS]

Options:
  -n, --nodes TEXT       Comma-separated list of nodes to monitor
  -i, --interval FLOAT   Refresh interval in seconds (default: 2.0)
  -w, --workers INTEGER  Maximum parallel SSH connections (default: 8)
  -c, --compact          Use compact table view (fits more nodes)
  -F, --no-fullscreen    Disable fullscreen mode (allows scrolling)
  -p, --processes        Show GPU processes with user info
  --version              Show the version and exit.
  --help                 Show this message and exit.
```

## 📸 Display Modes

### Default Mode (Full Screen Panels)

Each node displayed as a separate panel with GPU bars:

```
╭─────────────────────── ✓ visko-1 ───────────────────────╮
│ GPU0 ██████████████░░ 87%  │ ████████████░░ 74.9G/79.6G │
│ GPU1 ████████████░░░░ 78%  │ ██████████████ 75.0G/79.6G │
│ ...                                                      │
╰──────────────────────────────────────────────────────────╯
```

### Compact Mode (`-c`)

Table view with dual-column GPU layout for wide terminals:

```
  Node           #   Utl        Mem         │  #   Utl        Mem
 ─────────────────────────────────────────────────────────────────
  🔥 visko-1    0  100%  ████  74.9G/79.6G  │  1   78%  ███  75.0G/79.6G
               2  100%  ████  76.5G/79.6G  │  3   95%  ███  75.0G/79.6G
```

### Compact + Processes (`-c -p`)

Shows GPU processes grouped by user below the table:

```
  📋 visko-1 processes: user1[GPU 0,1]:15.2G │ user2[GPU 2,3]:30.5G
  📋 visko-3 processes: user3[GPU 0-7]:60.0G
```

## 📋 Requirements

- Python 3.8+
- SSH access to cluster nodes (passwordless recommended)
- `nvidia-smi` installed on each node
- (Optional) Slurm for auto-detection of nodes

## 📝 Changelog

### v0.1.0 (2026-01-09)

#### 🎉 Initial Release (`efed059`)
- Basic GPU monitoring via SSH
- Auto-detect Slurm nodes using `sinfo`
- Beautiful CLI dashboard with Rich library
- Parallel SSH queries for performance
- Real-time refresh with configurable interval

#### ✨ Compact & Scrollable Modes (`3e95c9a`)
- Added `-c / --compact` flag for table view
- Added `-F / --no-fullscreen` flag for terminal scrolling
- Compact mode uses less vertical space

#### 📊 Auto Multi-Column Layout (`5d8afe6`)
- Terminal width auto-detection
- Multi-column node panels in fullscreen mode (≥130 chars)
- Dual-GPU rows in compact mode (≥120 chars)
- Optimal use of horizontal screen space

#### 🔧 Layout Optimization (`9e0420f`)
- Reduced column widths for better density
- Optimized header labels (GPU → #, Util → Utl)
- Lowered dual-column threshold from 150 to 120 chars
- More compact progress bars (8→6 chars)

#### 👤 Process Display (`0d08cf1`)
- Added `-p / --processes` flag
- Shows GPU processes with user info
- Processes grouped by user with GPU IDs and memory
- Queries via `nvidia-smi --query-compute-apps`

## 📄 License

MIT License

# Node Monitor 🖥️

A beautiful CLI tool for monitoring GPU resources across Slurm cluster nodes.

![Python](https://img.shields.io/badge/python-3.8+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

- 🔍 **Auto-detect Slurm nodes** - Automatically discovers all nodes in your cluster
- 🎮 **GPU monitoring** - Shows utilization and memory usage for each GPU
- 🎨 **Beautiful CLI** - Colorful, live-updating dashboard using Rich
- ⚡ **Fast** - Parallel SSH queries for quick updates
- 🛠️ **Flexible** - Override nodes manually, customize refresh interval

## Installation

```bash
# Clone the repository
cd /path/to/node-monitor

# Install with pip
pip install -e .
```

## Usage

### Basic Usage

```bash
# Auto-detect Slurm nodes and start monitoring
node-monitor
```

### Specify Nodes Manually

```bash
# Monitor specific nodes
node-monitor --nodes visko-1,visko-2,visko-3
```

### Custom Refresh Interval

```bash
# Refresh every 5 seconds
node-monitor --interval 5
```

### All Options

```bash
node-monitor --help
```

```
Usage: node-monitor [OPTIONS]

  🖥️  GPU Cluster Monitor - Monitor GPU resources across Slurm nodes.

Options:
  -n, --nodes TEXT      Comma-separated list of nodes to monitor
  -i, --interval FLOAT  Refresh interval in seconds (default: 2.0)
  -w, --workers INT     Maximum parallel SSH connections (default: 8)
  --version             Show the version and exit.
  --help                Show this message and exit.
```

## Requirements

- Python 3.8+
- SSH access to cluster nodes
- `nvidia-smi` installed on each node
- (Optional) Slurm for auto-detection

## Screenshots

The dashboard shows:
- Per-node GPU utilization with color-coded bars
- Memory usage with visual progress indicators
- Cluster summary with total GPUs and average utilization
- Real-time updates with configurable refresh interval

## License

MIT License

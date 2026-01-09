"""Beautiful CLI visualization using Rich."""

from datetime import datetime
from typing import List

from rich.console import Console, Group
from rich.live import Live
from rich.panel import Panel
from rich.progress import BarColumn, Progress, TextColumn, TaskID
from rich.table import Table
from rich.text import Text
from rich.layout import Layout
from rich.style import Style
from rich import box

from node_monitor.monitor import NodeStatus, GPUInfo


def get_utilization_color(percent: float) -> str:
    """Get color based on utilization percentage."""
    if percent < 30:
        return "bright_green"
    elif percent < 60:
        return "yellow"
    elif percent < 85:
        return "orange1"
    else:
        return "red"


def get_memory_color(percent: float) -> str:
    """Get color based on memory usage percentage."""
    if percent < 50:
        return "bright_cyan"
    elif percent < 75:
        return "yellow"
    elif percent < 90:
        return "orange1"
    else:
        return "red"


def format_memory(mib: int) -> str:
    """Format memory in human-readable format."""
    if mib >= 1024:
        return f"{mib / 1024:.1f}G"
    return f"{mib}M"


def create_gpu_bar(gpu: GPUInfo, bar_width: int = 20) -> Text:
    """Create a colorful bar representation for a GPU."""
    util_color = get_utilization_color(gpu.utilization)
    mem_color = get_memory_color(gpu.memory_percent)
    
    # GPU utilization bar
    util_filled = int((gpu.utilization / 100) * bar_width)
    util_bar = "█" * util_filled + "░" * (bar_width - util_filled)
    
    # Memory bar
    mem_filled = int((gpu.memory_percent / 100) * bar_width)
    mem_bar = "█" * mem_filled + "░" * (bar_width - mem_filled)
    
    text = Text()
    text.append(f"  GPU {gpu.index} ", style="bold white")
    text.append("│ ", style="dim")
    
    # Utilization
    text.append("Util: ", style="dim white")
    text.append(util_bar, style=util_color)
    text.append(f" {gpu.utilization:3d}% ", style=f"bold {util_color}")
    
    text.append("│ ", style="dim")
    
    # Memory
    text.append("Mem: ", style="dim white")
    text.append(mem_bar, style=mem_color)
    mem_str = f" {format_memory(gpu.memory_used)}/{format_memory(gpu.memory_total)}"
    text.append(f"{mem_str:>12}", style=f"bold {mem_color}")
    
    return text


def create_node_panel(status: NodeStatus) -> Panel:
    """Create a panel for a single node."""
    if not status.is_online:
        # Offline node
        content = Text(f"  ⚠ {status.error}", style="red")
        return Panel(
            content,
            title=f"[bold red]✗ {status.hostname}[/]",
            border_style="red",
            box=box.ROUNDED,
        )
    
    if not status.gpus:
        content = Text("  No GPUs detected", style="dim yellow")
        return Panel(
            content,
            title=f"[bold yellow]? {status.hostname}[/]",
            border_style="yellow",
            box=box.ROUNDED,
        )
    
    # Create GPU bars
    lines = []
    for gpu in status.gpus:
        lines.append(create_gpu_bar(gpu))
    
    # Add summary line
    summary = Text()
    avg_util = status.avg_utilization
    util_color = get_utilization_color(avg_util)
    
    total_mem = status.total_memory
    used_mem = status.total_memory_used
    mem_percent = (used_mem / total_mem * 100) if total_mem > 0 else 0
    mem_color = get_memory_color(mem_percent)
    
    summary.append("\n  ─────────────────────────────────────────────────────────────────────────\n", style="dim")
    summary.append(f"  Σ {len(status.gpus)} GPUs ", style="bold white")
    summary.append("│ ", style="dim")
    summary.append(f"Avg Util: {avg_util:5.1f}% ", style=f"bold {util_color}")
    summary.append("│ ", style="dim")
    summary.append(f"Total Mem: {format_memory(used_mem)}/{format_memory(total_mem)} ", style=f"bold {mem_color}")
    summary.append(f"({mem_percent:.1f}%)", style=f"{mem_color}")
    
    lines.append(summary)
    
    content = Text("\n").join(lines)
    
    # Determine overall status color
    if avg_util > 80:
        border_color = "red"
        status_icon = "🔥"
    elif avg_util > 50:
        border_color = "yellow"
        status_icon = "⚡"
    else:
        border_color = "green"
        status_icon = "✓"
    
    return Panel(
        content,
        title=f"[bold {border_color}]{status_icon} {status.hostname}[/]",
        border_style=border_color,
        box=box.ROUNDED,
        padding=(0, 1),
    )


def create_dashboard(nodes: List[NodeStatus], refresh_interval: float) -> Panel:
    """Create the complete dashboard view."""
    # Header with timestamp
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    # Calculate cluster summary
    total_gpus = sum(node.total_gpus for node in nodes if node.is_online)
    online_nodes = sum(1 for node in nodes if node.is_online)
    offline_nodes = sum(1 for node in nodes if not node.is_online)
    
    total_util = 0
    total_mem_used = 0
    total_mem = 0
    
    for node in nodes:
        if node.is_online:
            total_mem_used += node.total_memory_used
            total_mem += node.total_memory
            total_util += sum(gpu.utilization for gpu in node.gpus)
    
    avg_util = total_util / total_gpus if total_gpus > 0 else 0
    avg_mem = (total_mem_used / total_mem * 100) if total_mem > 0 else 0
    
    # Create header
    header = Text()
    header.append("╭─────────────────────────────────────────────────────────────────────────────────────╮\n", style="cyan")
    header.append("│  ", style="cyan")
    header.append("🖥️  GPU Cluster Monitor", style="bold bright_white")
    header.append(f"   │   🕐 {now}", style="white")
    header.append(f"   │   🔄 Refresh: {refresh_interval}s", style="dim white")
    header.append("  │\n", style="cyan")
    header.append("├─────────────────────────────────────────────────────────────────────────────────────┤\n", style="cyan")
    header.append("│  ", style="cyan")
    header.append(f"📊 Nodes: ", style="white")
    header.append(f"{online_nodes} online", style="bold green")
    if offline_nodes > 0:
        header.append(f" / {offline_nodes} offline", style="bold red")
    header.append(f"   │   🎮 Total GPUs: ", style="white")
    header.append(f"{total_gpus}", style="bold bright_cyan")
    header.append(f"   │   ", style="white")
    
    util_color = get_utilization_color(avg_util)
    header.append(f"⚡ Avg Util: ", style="white")
    header.append(f"{avg_util:.1f}%", style=f"bold {util_color}")
    
    mem_color = get_memory_color(avg_mem)
    header.append(f"   │   💾 Mem: ", style="white")
    header.append(f"{format_memory(total_mem_used)}/{format_memory(total_mem)}", style=f"bold {mem_color}")
    header.append("  │\n", style="cyan")
    header.append("╰─────────────────────────────────────────────────────────────────────────────────────╯", style="cyan")
    
    # Create node panels
    panels = [create_node_panel(node) for node in nodes]
    
    # Combine all elements
    elements = [header, Text("")]
    elements.extend(panels)
    elements.append(Text("\n  [dim]Press Ctrl+C to exit[/]", style="dim italic"))
    
    return Group(*elements)


class DashboardDisplay:
    """Manages the live dashboard display."""
    
    def __init__(self, refresh_interval: float = 2.0):
        self.console = Console()
        self.refresh_interval = refresh_interval
        self.live: Optional[Live] = None
    
    def start(self):
        """Start the live display."""
        self.live = Live(
            console=self.console,
            refresh_per_second=4,
            screen=True,
        )
        self.live.start()
    
    def stop(self):
        """Stop the live display."""
        if self.live:
            self.live.stop()
    
    def update(self, nodes: List[NodeStatus]):
        """Update the display with new node data."""
        if self.live:
            dashboard = create_dashboard(nodes, self.refresh_interval)
            self.live.update(dashboard)
    
    def __enter__(self):
        self.start()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.stop()
        return False

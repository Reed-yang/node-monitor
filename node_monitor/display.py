"""Beautiful CLI visualization using Rich."""

from datetime import datetime
from typing import List, Optional

from rich.console import Console, Group
from rich.live import Live
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.columns import Columns
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


def create_compact_bar(percent: float, width: int = 10) -> str:
    """Create a compact progress bar."""
    filled = int((percent / 100) * width)
    return "█" * filled + "░" * (width - filled)


def create_gpu_bar(gpu: GPUInfo, bar_width: int = 15) -> Text:
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
    text.append(f"GPU{gpu.index} ", style="bold white")
    text.append(util_bar, style=util_color)
    text.append(f" {gpu.utilization:3d}%", style=f"bold {util_color}")
    text.append(" │ ", style="dim")
    text.append(mem_bar, style=mem_color)
    mem_str = f" {format_memory(gpu.memory_used)}/{format_memory(gpu.memory_total)}"
    text.append(mem_str, style=f"bold {mem_color}")
    
    return text


def create_node_panel(status: NodeStatus, width: Optional[int] = None) -> Panel:
    """Create a panel for a single node."""
    panel_width = width if width else None
    
    if not status.is_online:
        content = Text(f"⚠ {status.error[:40] if status.error else 'Offline'}", style="red")
        return Panel(
            content,
            title=f"[bold red]✗ {status.hostname}[/]",
            border_style="red",
            box=box.ROUNDED,
            width=panel_width,
        )
    
    if not status.gpus:
        content = Text("No GPUs detected", style="dim yellow")
        return Panel(
            content,
            title=f"[bold yellow]? {status.hostname}[/]",
            border_style="yellow",
            box=box.ROUNDED,
            width=panel_width,
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
    
    summary.append("────────────────────────────────────────────────────────\n", style="dim")
    summary.append(f"Σ {len(status.gpus)} GPUs", style="bold white")
    summary.append(" │ ", style="dim")
    summary.append(f"Util: {avg_util:.0f}%", style=f"bold {util_color}")
    summary.append(" │ ", style="dim")
    summary.append(f"Mem: {format_memory(used_mem)}/{format_memory(total_mem)}", style=f"bold {mem_color}")
    
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
        width=panel_width,
    )


def create_header(nodes: List[NodeStatus], refresh_interval: float, terminal_width: int) -> Text:
    """Create the dashboard header."""
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
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
    
    util_color = get_utilization_color(avg_util)
    mem_color = get_memory_color(avg_mem)
    
    header = Text()
    header.append("🖥️  GPU Cluster Monitor", style="bold bright_white")
    header.append(f"  │  🕐 {now}", style="white")
    header.append(f"  │  🔄 {refresh_interval}s", style="dim")
    header.append(f"  │  📊 {online_nodes}", style="green")
    if offline_nodes > 0:
        header.append(f"/{offline_nodes}", style="red")
    header.append(" nodes", style="white")
    header.append(f"  │  🎮 {total_gpus} GPUs", style="bright_cyan")
    header.append(f"  │  ⚡ {avg_util:.0f}%", style=f"bold {util_color}")
    header.append(f"  │  💾 {format_memory(total_mem_used)}/{format_memory(total_mem)}", style=f"bold {mem_color}")
    
    return header


def create_dashboard(nodes: List[NodeStatus], refresh_interval: float, terminal_width: int = 120) -> Group:
    """Create the complete dashboard view with auto-column layout."""
    header = create_header(nodes, refresh_interval, terminal_width)
    
    # Calculate how many columns we can fit
    # Each panel needs about 65 chars minimum for good display
    min_panel_width = 65
    num_columns = max(1, terminal_width // min_panel_width)
    
    # Create node panels
    panel_width = (terminal_width - 4) // num_columns if num_columns > 1 else None
    panels = [create_node_panel(node, panel_width) for node in nodes]
    
    # Arrange panels in rows
    elements = [header, Text("")]
    
    if num_columns > 1:
        # Multi-column layout
        for i in range(0, len(panels), num_columns):
            row_panels = panels[i:i + num_columns]
            elements.append(Columns(row_panels, equal=True, expand=True))
    else:
        # Single column layout
        elements.extend(panels)
    
    elements.append(Text("\n  Press Ctrl+C to exit", style="dim italic"))
    
    return Group(*elements)


def create_compact_table(nodes: List[NodeStatus], refresh_interval: float, terminal_width: int = 120) -> Group:
    """Create a compact table view with multi-column GPU display when width allows."""
    header = create_header(nodes, refresh_interval, terminal_width)
    
    # Calculate if we can fit 2 GPUs per row
    # 优化策略：由于我们压缩了单列的宽度，Dual Column (双列) 模式的触发阈值可以降低
    # 原单列约需 75 字符，优化后约需 55 字符。双列现在只需要约 110-120 字符即可显示
    dual_column = terminal_width >= 120  # 从 150 降到 120，更容易触发双列显示
    
    # 定义超参数 (Hyperparameters)
    BAR_WIDTH = 6          #  将进度条从 8/10 减少到 6，视觉更紧凑
    GPU_COL_WIDTH = 2      #  GPU ID 只需要很窄的空间
    UTIL_COL_WIDTH = 4     #  "100%" 刚好4个字符
    MEM_COL_WIDTH = 11     #  根据截图 74.9G/79.6G 刚好紧凑放下

    table = Table(
        box=box.SIMPLE_HEAD,
        show_header=True,
        header_style="bold cyan",
        expand=True,
        padding=(0, 1), # 保持左右各1个空格的padding，避免太拥挤
    )
    
    if dual_column:
        # Dual-column GPU layout
        table.add_column("Node", style="bold white", no_wrap=True, min_width=10) # 稍微减小 min_width
        table.add_column("#", justify="center", width=GPU_COL_WIDTH)       #  "#"
        table.add_column("Utl", justify="right", width=UTIL_COL_WIDTH)     #  "Utl"
        table.add_column("", width=BAR_WIDTH)                              #  ""
        table.add_column("Mem", justify="right", width=MEM_COL_WIDTH)      # shorter Header
        
        table.add_column("│", justify="center", width=1, style="dim") 
        
        table.add_column("#", justify="center", width=GPU_COL_WIDTH)
        table.add_column("Utl", justify="right", width=UTIL_COL_WIDTH)
        table.add_column("", width=BAR_WIDTH)
        table.add_column("Mem", justify="right", width=MEM_COL_WIDTH)
    else:
        # Single-column GPU layout
        table.add_column("Node", style="bold white", no_wrap=True, min_width=12)
        table.add_column("#", justify="center", width=GPU_COL_WIDTH)       #  Header: GPU -> #
        table.add_column("Util", justify="right", width=UTIL_COL_WIDTH)
        table.add_column("", width=BAR_WIDTH)                              #  Header: Util Bar -> ""
        table.add_column("Memory", justify="right", width=MEM_COL_WIDTH)
        table.add_column("", width=BAR_WIDTH)                              #  Header: Mem Bar -> ""
        table.add_column("%", justify="right", width=4)                    #  Header: Mem% -> %
    
    for node in nodes:
        if not node.is_online:
            if dual_column:
                table.add_row(
                    Text(f"✗ {node.hostname}", style="red"),
                    "-", "-", "-", 
                    Text(node.error[:20] if node.error else "Offline", style="dim red"),
                    "", "", "", "", ""
                )
            else:
                table.add_row(
                    Text(f"✗ {node.hostname}", style="red"),
                    "-", "-", "-",
                    Text(node.error[:25] if node.error else "Offline", style="dim red"),
                    "-", "-"
                )
            continue
        
        if not node.gpus:
            if dual_column:
                table.add_row(
                    Text(f"? {node.hostname}", style="yellow"),
                    "-", "-", "-",
                    Text("No GPUs", style="dim yellow"),
                    "", "", "", "", ""
                )
            else:
                table.add_row(
                    Text(f"? {node.hostname}", style="yellow"),
                    "-", "-", "-",
                    Text("No GPUs", style="dim yellow"),
                    "-", "-"
                )
            continue
        
        # Get status icon
        avg = node.avg_utilization
        if avg > 80:
            icon = "🔥"
        elif avg > 50:
            icon = "⚡"
        else:
            icon = "✓"
        
        if dual_column:
            # Two GPUs per row
            gpus = node.gpus
            for i in range(0, len(gpus), 2):
                gpu1 = gpus[i]
                gpu2 = gpus[i + 1] if i + 1 < len(gpus) else None
                
                node_name = f"{icon} {node.hostname}" if i == 0 else ""
                
                util_color1 = get_utilization_color(gpu1.utilization)
                mem_color1 = get_memory_color(gpu1.memory_percent)
                
                row = [
                    node_name,
                    str(gpu1.index),
                    Text(f"{gpu1.utilization}%", style=f"bold {util_color1}"),
                    Text(create_compact_bar(gpu1.utilization, 8), style=util_color1),
                    f"{format_memory(gpu1.memory_used)}/{format_memory(gpu1.memory_total)}",
                    Text("│", style="dim"),
                ]
                
                if gpu2:
                    util_color2 = get_utilization_color(gpu2.utilization)
                    mem_color2 = get_memory_color(gpu2.memory_percent)
                    row.extend([
                        str(gpu2.index),
                        Text(f"{gpu2.utilization}%", style=f"bold {util_color2}"),
                        Text(create_compact_bar(gpu2.utilization, 8), style=util_color2),
                        f"{format_memory(gpu2.memory_used)}/{format_memory(gpu2.memory_total)}",
                    ])
                else:
                    row.extend(["", "", "", ""])
                
                table.add_row(*row)
        else:
            # One GPU per row
            for i, gpu in enumerate(node.gpus):
                util_color = get_utilization_color(gpu.utilization)
                mem_color = get_memory_color(gpu.memory_percent)
                
                node_name = f"{icon} {node.hostname}" if i == 0 else ""
                
                table.add_row(
                    node_name,
                    str(gpu.index),
                    Text(f"{gpu.utilization}%", style=f"bold {util_color}"),
                    Text(create_compact_bar(gpu.utilization, 8), style=util_color),
                    f"{format_memory(gpu.memory_used)}/{format_memory(gpu.memory_total)}",
                    Text(create_compact_bar(gpu.memory_percent, 8), style=mem_color),
                    Text(f"{gpu.memory_percent:.0f}%", style=f"bold {mem_color}"),
                )
    
    footer = Text("\n  Press Ctrl+C to exit", style="dim italic")
    
    return Group(header, Text(""), table, footer)


class DashboardDisplay:
    """Manages the live dashboard display."""
    
    def __init__(self, refresh_interval: float = 2.0, compact: bool = False, fullscreen: bool = True):
        self.console = Console()
        self.refresh_interval = refresh_interval
        self.compact = compact
        self.fullscreen = fullscreen
        self.live: Optional[Live] = None
    
    def start(self):
        """Start the live display."""
        self.live = Live(
            console=self.console,
            refresh_per_second=4,
            screen=self.fullscreen,
        )
        self.live.start()
    
    def stop(self):
        """Stop the live display."""
        if self.live:
            self.live.stop()
    
    def update(self, nodes: List[NodeStatus]):
        """Update the display with new node data."""
        if self.live:
            # Get current terminal width
            width = self.console.width
            
            if self.compact:
                dashboard = create_compact_table(nodes, self.refresh_interval, width)
            else:
                dashboard = create_dashboard(nodes, self.refresh_interval, width)
            self.live.update(dashboard)
    
    def __enter__(self):
        self.start()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.stop()
        return False

"""CLI entry point for node-monitor."""

import signal
import sys
import time
from typing import Optional

import click
from rich.console import Console

from node_monitor import __version__
from node_monitor.slurm import get_slurm_nodes, parse_node_list
from node_monitor.monitor import query_all_nodes
from node_monitor.display import DashboardDisplay


console = Console()


@click.command()
@click.option(
    "--nodes", "-n",
    type=str,
    default=None,
    help="Comma-separated list of nodes to monitor (e.g., 'visko-1,visko-2'). "
         "If not specified, auto-detects from Slurm.",
)
@click.option(
    "--interval", "-i",
    type=float,
    default=2.0,
    help="Refresh interval in seconds (default: 2.0)",
)
@click.option(
    "--workers", "-w",
    type=int,
    default=8,
    help="Maximum parallel SSH connections (default: 8)",
)
@click.option(
    "--compact", "-c",
    is_flag=True,
    default=False,
    help="Use compact table view (fits more nodes on screen)",
)
@click.option(
    "--no-fullscreen", "-F",
    is_flag=True,
    default=False,
    help="Disable fullscreen mode (allows terminal scrolling)",
)
@click.option(
    "--processes", "-p",
    is_flag=True,
    default=False,
    help="Show GPU processes with user info (requires -c compact mode)",
)
@click.version_option(version=__version__, prog_name="node-monitor")
def main(nodes: Optional[str], interval: float, workers: int, compact: bool, no_fullscreen: bool, processes: bool):
    """
    🖥️  GPU Cluster Monitor - Monitor GPU resources across Slurm nodes.
    
    Displays a beautiful, live-updating dashboard showing GPU utilization
    and memory usage for each node in your cluster.
    
    \b
    Examples:
        node-monitor                     # Auto-detect Slurm nodes
        node-monitor -n visko-1,visko-2  # Monitor specific nodes
        node-monitor -i 5                # Refresh every 5 seconds
        node-monitor -c                  # Compact table view
        node-monitor -c -p               # Compact + show processes
    """
    # Determine nodes to monitor
    if nodes:
        node_list = parse_node_list(nodes)
        console.print(f"[cyan]📋 Monitoring specified nodes:[/] {', '.join(node_list)}")
    else:
        try:
            console.print("[cyan]🔍 Detecting Slurm nodes...[/]")
            node_list = get_slurm_nodes()
            console.print(f"[green]✓ Found {len(node_list)} nodes:[/] {', '.join(node_list)}")
        except RuntimeError as e:
            console.print(f"[red]✗ Error: {e}[/]")
            console.print("[yellow]💡 Tip: Use --nodes option to specify nodes manually[/]")
            sys.exit(1)
    
    if not node_list:
        console.print("[red]✗ No nodes to monitor[/]")
        sys.exit(1)
    
    # Process display requires compact mode
    if processes and not compact:
        console.print("[yellow]💡 Note: --processes requires --compact mode, enabling compact mode[/]")
        compact = True
    
    mode_info = []
    if compact:
        mode_info.append("compact")
    if processes:
        mode_info.append("processes")
    if no_fullscreen:
        mode_info.append("scrollable")
    mode_str = f" ({', '.join(mode_info)})" if mode_info else ""
    
    console.print(f"[cyan]🚀 Starting monitor{mode_str} (refresh every {interval}s)...[/]")
    time.sleep(1)  # Brief pause before entering full-screen mode
    
    # Set up graceful shutdown
    running = True
    
    def signal_handler(sig, frame):
        nonlocal running
        running = False
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Main monitoring loop
    try:
        with DashboardDisplay(
            refresh_interval=interval,
            compact=compact,
            fullscreen=not no_fullscreen,
            show_processes=processes,
        ) as display:
            while running:
                # Query all nodes in parallel
                statuses = query_all_nodes(node_list, max_workers=workers, show_processes=processes)
                
                # Update display
                display.update(statuses)
                
                # Wait for next refresh
                time.sleep(interval)
    except KeyboardInterrupt:
        pass
    finally:
        console.print("\n[cyan]👋 Monitor stopped. Goodbye![/]")


if __name__ == "__main__":
    main()

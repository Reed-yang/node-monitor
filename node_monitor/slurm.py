"""Slurm node detection utilities."""

import subprocess
from typing import List, Optional


def get_slurm_nodes() -> List[str]:
    """
    Detect all available Slurm nodes using sinfo command.
    
    Returns:
        List of node hostnames.
    
    Raises:
        RuntimeError: If Slurm is not available or sinfo fails.
    """
    try:
        # Use sinfo to get all node names
        # -h: no header, -o "%n": just node names
        result = subprocess.run(
            ["sinfo", "-h", "-o", "%n"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        
        if result.returncode != 0:
            raise RuntimeError(f"sinfo command failed: {result.stderr}")
        
        # Parse output: one node per line, may have duplicates
        nodes = set()
        for line in result.stdout.strip().split("\n"):
            node = line.strip()
            if node:
                nodes.add(node)
        
        return sorted(nodes)
    
    except FileNotFoundError:
        raise RuntimeError(
            "Slurm is not installed or sinfo is not in PATH. "
            "Use --nodes option to specify nodes manually."
        )
    except subprocess.TimeoutExpired:
        raise RuntimeError("sinfo command timed out")


def parse_node_list(node_string: str) -> List[str]:
    """
    Parse a comma-separated list of nodes.
    
    Args:
        node_string: Comma-separated node names, e.g., "visko-1,visko-2,visko-3"
    
    Returns:
        List of node hostnames.
    """
    nodes = []
    for node in node_string.split(","):
        node = node.strip()
        if node:
            nodes.append(node)
    return nodes

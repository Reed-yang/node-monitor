"""GPU monitoring via SSH."""

import subprocess
from dataclasses import dataclass
from typing import List, Optional
from concurrent.futures import ThreadPoolExecutor, as_completed


@dataclass
class GPUInfo:
    """Information about a single GPU."""
    index: int
    utilization: int  # GPU utilization percentage (0-100)
    memory_used: int  # Memory used in MiB
    memory_total: int  # Total memory in MiB
    
    @property
    def memory_percent(self) -> float:
        """Memory usage as a percentage."""
        if self.memory_total == 0:
            return 0.0
        return (self.memory_used / self.memory_total) * 100


@dataclass
class NodeStatus:
    """Status of a node including all its GPUs."""
    hostname: str
    gpus: List[GPUInfo]
    error: Optional[str] = None
    
    @property
    def is_online(self) -> bool:
        """Check if the node is reachable."""
        return self.error is None
    
    @property
    def total_gpus(self) -> int:
        """Total number of GPUs on this node."""
        return len(self.gpus)
    
    @property
    def avg_utilization(self) -> float:
        """Average GPU utilization across all GPUs."""
        if not self.gpus:
            return 0.0
        return sum(gpu.utilization for gpu in self.gpus) / len(self.gpus)
    
    @property
    def total_memory_used(self) -> int:
        """Total memory used across all GPUs in MiB."""
        return sum(gpu.memory_used for gpu in self.gpus)
    
    @property
    def total_memory(self) -> int:
        """Total memory across all GPUs in MiB."""
        return sum(gpu.memory_total for gpu in self.gpus)


def query_node_gpus(hostname: str, timeout: int = 10) -> NodeStatus:
    """
    Query GPU information from a node via SSH.
    
    Args:
        hostname: The hostname to SSH into.
        timeout: SSH command timeout in seconds.
    
    Returns:
        NodeStatus with GPU information or error.
    """
    nvidia_smi_cmd = (
        "nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total "
        "--format=csv,noheader,nounits"
    )
    
    try:
        result = subprocess.run(
            [
                "ssh",
                "-o", "StrictHostKeyChecking=no",
                "-o", "ConnectTimeout=5",
                "-o", "BatchMode=yes",
                hostname,
                nvidia_smi_cmd,
            ],
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        
        if result.returncode != 0:
            error_msg = result.stderr.strip() or f"SSH command failed with code {result.returncode}"
            return NodeStatus(hostname=hostname, gpus=[], error=error_msg)
        
        # Parse CSV output
        gpus = []
        for line in result.stdout.strip().split("\n"):
            if not line.strip():
                continue
            
            parts = [p.strip() for p in line.split(",")]
            if len(parts) >= 4:
                try:
                    gpu = GPUInfo(
                        index=int(parts[0]),
                        utilization=int(parts[1]),
                        memory_used=int(parts[2]),
                        memory_total=int(parts[3]),
                    )
                    gpus.append(gpu)
                except ValueError:
                    continue
        
        return NodeStatus(hostname=hostname, gpus=gpus)
    
    except subprocess.TimeoutExpired:
        return NodeStatus(hostname=hostname, gpus=[], error="Connection timed out")
    except Exception as e:
        return NodeStatus(hostname=hostname, gpus=[], error=str(e))


def query_all_nodes(hostnames: List[str], max_workers: int = 8) -> List[NodeStatus]:
    """
    Query GPU information from multiple nodes in parallel.
    
    Args:
        hostnames: List of hostnames to query.
        max_workers: Maximum number of parallel SSH connections.
    
    Returns:
        List of NodeStatus objects, one for each hostname.
    """
    results = []
    
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        future_to_host = {
            executor.submit(query_node_gpus, host): host 
            for host in hostnames
        }
        
        for future in as_completed(future_to_host):
            results.append(future.result())
    
    # Sort by hostname to maintain consistent ordering
    results.sort(key=lambda x: x.hostname)
    return results

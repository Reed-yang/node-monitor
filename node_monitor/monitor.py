"""GPU monitoring via SSH."""

import subprocess
from dataclasses import dataclass, field
from typing import List, Optional, Dict
from concurrent.futures import ThreadPoolExecutor, as_completed


@dataclass
class GPUProcess:
    """Information about a GPU process."""

    pid: int
    user: str
    gpu_index: int
    memory_used: int  # Memory in MiB
    command: str  # Process command (shortened)


@dataclass
class GPUInfo:
    """Information about a single GPU."""

    index: int
    utilization: int  # GPU utilization percentage (0-100)
    memory_used: int  # Memory used in MiB
    memory_total: int  # Total memory in MiB
    processes: List[GPUProcess] = field(default_factory=list)

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

    @property
    def all_processes(self) -> List[GPUProcess]:
        """All processes across all GPUs."""
        procs = []
        for gpu in self.gpus:
            procs.extend(gpu.processes)
        return procs


def query_node_gpus(
    hostname: str, timeout: int = 10, show_processes: bool = False, debug: bool = False
) -> NodeStatus:
    """
    Query GPU information from a node via SSH.

    Args:
        hostname: The hostname to SSH into.
        timeout: SSH command timeout in seconds.
        show_processes: Whether to also query GPU processes.
        debug: Whether to include verbose error information.

    Returns:
        NodeStatus with GPU information or error.
    """
    # Build command: GPU stats and optionally process info
    nvidia_smi_cmd = (
        "nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total "
        "--format=csv,noheader,nounits"
    )

    if show_processes:
        # Also query compute processes
        nvidia_smi_cmd += (
            " && echo '---PROCESSES---' && "
            "nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory,process_name "
            "--format=csv,noheader,nounits 2>/dev/null || true"
            " && echo '---GPU_UUID_MAP---' && "
            "nvidia-smi --query-gpu=index,uuid --format=csv,noheader 2>/dev/null || true"
            " && echo '---USERS---' && "
            "ps -eo pid,user --no-headers 2>/dev/null || true"
        )

    # Build SSH command with optional verbose flag for debugging
    ssh_args = [
        "ssh",
        "-o",
        "StrictHostKeyChecking=no",
        "-o",
        "ConnectTimeout=5",
        "-o",
        "BatchMode=yes",
    ]
    if debug:
        ssh_args.append("-v")  # Verbose SSH output for debugging
    ssh_args.extend([hostname, nvidia_smi_cmd])

    try:
        result = subprocess.run(
            ssh_args,
            capture_output=True,
            text=True,
            timeout=timeout,
        )

        if result.returncode != 0:
            if debug:
                # Include both stderr and stdout for debugging
                error_msg = f"SSH failed (code {result.returncode})\nSTDERR: {result.stderr.strip()}\nSTDOUT: {result.stdout.strip()}"
            else:
                error_msg = (
                    result.stderr.strip()
                    or f"SSH command failed with code {result.returncode}"
                )
            return NodeStatus(hostname=hostname, gpus=[], error=error_msg)

        output = result.stdout.strip()

        # Parse output
        if show_processes and "---PROCESSES---" in output:
            parts = output.split("---PROCESSES---")
            gpu_output = parts[0].strip()
            rest = parts[1] if len(parts) > 1 else ""

            # Parse process info
            process_output = ""
            uuid_map_output = ""
            users_output = ""

            if "---GPU_UUID_MAP---" in rest:
                proc_parts = rest.split("---GPU_UUID_MAP---")
                process_output = proc_parts[0].strip()
                rest2 = proc_parts[1] if len(proc_parts) > 1 else ""

                if "---USERS---" in rest2:
                    uuid_parts = rest2.split("---USERS---")
                    uuid_map_output = uuid_parts[0].strip()
                    users_output = uuid_parts[1].strip() if len(uuid_parts) > 1 else ""

            gpus = _parse_gpu_output(gpu_output)

            # Build UUID to GPU index map
            uuid_to_idx = {}
            for line in uuid_map_output.split("\n"):
                if line.strip():
                    p = [x.strip() for x in line.split(",")]
                    if len(p) >= 2:
                        try:
                            uuid_to_idx[p[1]] = int(p[0])
                        except ValueError:
                            pass

            # Build PID to user map
            pid_to_user = {}
            for line in users_output.split("\n"):
                if line.strip():
                    p = line.split()
                    if len(p) >= 2:
                        try:
                            pid_to_user[int(p[0])] = p[1]
                        except ValueError:
                            pass

            # Parse processes and assign to GPUs
            gpu_dict = {gpu.index: gpu for gpu in gpus}
            for line in process_output.split("\n"):
                if line.strip():
                    p = [x.strip() for x in line.split(",")]
                    if len(p) >= 4:
                        try:
                            gpu_uuid = p[0]
                            pid = int(p[1])
                            mem = int(p[2]) if p[2] else 0
                            cmd = p[3].split("/")[-1][:20]  # Get basename, truncate

                            gpu_idx = uuid_to_idx.get(gpu_uuid, -1)
                            user = pid_to_user.get(pid, "?")

                            if gpu_idx in gpu_dict:
                                proc = GPUProcess(
                                    pid=pid,
                                    user=user[:8],  # Truncate username
                                    gpu_index=gpu_idx,
                                    memory_used=mem,
                                    command=cmd,
                                )
                                gpu_dict[gpu_idx].processes.append(proc)
                        except (ValueError, IndexError):
                            pass

            return NodeStatus(hostname=hostname, gpus=gpus)
        else:
            gpus = _parse_gpu_output(output)
            return NodeStatus(hostname=hostname, gpus=gpus)

    except subprocess.TimeoutExpired:
        error_msg = (
            f"Connection timed out after {timeout}s"
            if debug
            else "Connection timed out"
        )
        return NodeStatus(hostname=hostname, gpus=[], error=error_msg)
    except Exception as e:
        error_msg = f"Exception: {type(e).__name__}: {str(e)}" if debug else str(e)
        return NodeStatus(hostname=hostname, gpus=[], error=error_msg)


def _parse_gpu_output(output: str) -> List[GPUInfo]:
    """Parse nvidia-smi GPU query output."""
    gpus = []
    for line in output.split("\n"):
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
    return gpus


def query_all_nodes(
    hostnames: List[str],
    max_workers: int = 8,
    show_processes: bool = False,
    debug: bool = False,
) -> List[NodeStatus]:
    """
    Query GPU information from multiple nodes in parallel.

    Args:
        hostnames: List of hostnames to query.
        max_workers: Maximum number of parallel SSH connections.
        show_processes: Whether to also query GPU processes.
        debug: Whether to include verbose error information.

    Returns:
        List of NodeStatus objects, one for each hostname.
    """
    results = []

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        future_to_host = {
            executor.submit(query_node_gpus, host, 10, show_processes, debug): host
            for host in hostnames
        }

        for future in as_completed(future_to_host):
            results.append(future.result())

    # Sort by hostname to maintain consistent ordering
    results.sort(key=lambda x: x.hostname)
    return results

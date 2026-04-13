package ssh

import (
	"testing"
)

func TestParseGPUOutput(t *testing.T) {
	output := "0, 87, 74900, 79600\n1, 78, 75000, 79600\n2, 100, 76500, 79600\n"
	gpus := parseGPUOutput(output)

	if len(gpus) != 3 {
		t.Fatalf("got %d GPUs, want 3", len(gpus))
	}

	if gpus[0].Index != 0 || gpus[0].Utilization != 87 || gpus[0].MemoryUsed != 74900 {
		t.Errorf("GPU0 = %+v", gpus[0])
	}
	if gpus[2].Utilization != 100 {
		t.Errorf("GPU2 utilization = %d, want 100", gpus[2].Utilization)
	}
}

func TestParseGPUOutput_Empty(t *testing.T) {
	gpus := parseGPUOutput("")
	if len(gpus) != 0 {
		t.Errorf("got %d GPUs, want 0", len(gpus))
	}
}

func TestParseGPUOutput_MalformedLine(t *testing.T) {
	output := "0, 87, 74900, 79600\nbadline\n1, 50, 1000, 2000\n"
	gpus := parseGPUOutput(output)
	if len(gpus) != 2 {
		t.Errorf("got %d GPUs, want 2 (skipping malformed)", len(gpus))
	}
}

func TestParseDetailOutput(t *testing.T) {
	output := `0, 87, 74900, 79600
1, 78, 75000, 79600
---PROCESSES---
GPU-uuid-aaa, 1234, 5000, python
GPU-uuid-bbb, 5678, 3000, train.py
---GPU_UUID_MAP---
0, GPU-uuid-aaa
1, GPU-uuid-bbb
---USERS---
1234 alice
5678 bob
---SYSTEM---
1.23 2.34 3.45 2/100 12345
              total        used        free      shared  buff/cache   available
Mem:    540000000000 270000000000 135000000000  1000000 134000000000 268000000000
NVIDIA UNIX x86_64 Kernel Module  535.129.03  Tue Oct 17 11:42:00 UTC 2023`

	result := parseDetailOutput(output)

	if len(result.GPUs) != 2 {
		t.Fatalf("got %d GPUs, want 2", len(result.GPUs))
	}

	// Check processes were assigned to GPUs
	gpu0Procs := result.GPUs[0].Processes
	if len(gpu0Procs) != 1 || gpu0Procs[0].User != "alice" {
		t.Errorf("GPU0 processes = %+v, want 1 process by alice", gpu0Procs)
	}

	gpu1Procs := result.GPUs[1].Processes
	if len(gpu1Procs) != 1 || gpu1Procs[0].User != "bob" {
		t.Errorf("GPU1 processes = %+v, want 1 process by bob", gpu1Procs)
	}

	// Check system info
	if result.System == nil {
		t.Fatal("System info is nil")
	}
	if result.System.LoadAvg1 < 1.22 || result.System.LoadAvg1 > 1.24 {
		t.Errorf("LoadAvg1 = %f, want ~1.23", result.System.LoadAvg1)
	}
	if result.System.DriverVersion != "535.129.03" {
		t.Errorf("DriverVersion = %q, want 535.129.03", result.System.DriverVersion)
	}
}

func TestParseDetailOutput_NoProcesses(t *testing.T) {
	output := `0, 87, 74900, 79600
---PROCESSES---

---GPU_UUID_MAP---
0, GPU-uuid-aaa
---USERS---
1234 root
---SYSTEM---
0.50 0.60 0.70 1/50 999
              total        used        free      shared  buff/cache   available
Mem:    540000000000 100000000000 400000000000  1000000  40000000000 430000000000
N/A`

	result := parseDetailOutput(output)
	if len(result.GPUs) != 1 {
		t.Fatalf("got %d GPUs, want 1", len(result.GPUs))
	}
	if len(result.GPUs[0].Processes) != 0 {
		t.Errorf("expected 0 processes, got %d", len(result.GPUs[0].Processes))
	}
	if result.System.DriverVersion != "N/A" {
		t.Errorf("DriverVersion = %q, want N/A", result.System.DriverVersion)
	}
}

package model

import "testing"

func TestGPUInfo_MemoryPercent(t *testing.T) {
	tests := []struct {
		name     string
		gpu      GPUInfo
		expected float64
	}{
		{"normal usage", GPUInfo{Index: 0, Utilization: 87, MemoryUsed: 74900, MemoryTotal: 79600}, 94.09},
		{"zero total", GPUInfo{Index: 0, Utilization: 0, MemoryUsed: 0, MemoryTotal: 0}, 0.0},
		{"empty", GPUInfo{Index: 0, Utilization: 0, MemoryUsed: 0, MemoryTotal: 1024}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gpu.MemoryPercent()
			// Allow 0.1% tolerance
			if got < tt.expected-0.1 || got > tt.expected+0.1 {
				t.Errorf("MemoryPercent() = %v, want ~%v", got, tt.expected)
			}
		})
	}
}

func TestNodeStatus_Aggregates(t *testing.T) {
	node := NodeStatus{
		Hostname: "visko-1",
		GPUs: []GPUInfo{
			{Index: 0, Utilization: 80, MemoryUsed: 40000, MemoryTotal: 80000},
			{Index: 1, Utilization: 60, MemoryUsed: 20000, MemoryTotal: 80000},
		},
	}

	if node.TotalGPUs() != 2 {
		t.Errorf("TotalGPUs() = %d, want 2", node.TotalGPUs())
	}
	if node.AvgUtilization() != 70.0 {
		t.Errorf("AvgUtilization() = %f, want 70.0", node.AvgUtilization())
	}
	if node.TotalMemoryUsed() != 60000 {
		t.Errorf("TotalMemoryUsed() = %d, want 60000", node.TotalMemoryUsed())
	}
	if node.TotalMemory() != 160000 {
		t.Errorf("TotalMemory() = %d, want 160000", node.TotalMemory())
	}
	if !node.IsOnline() {
		t.Error("IsOnline() = false, want true")
	}
}

func TestNodeStatus_Offline(t *testing.T) {
	errMsg := "Connection timed out"
	node := NodeStatus{
		Hostname: "visko-2",
		GPUs:     nil,
		Error:    &errMsg,
	}
	if node.IsOnline() {
		t.Error("IsOnline() = true, want false")
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		mib      int
		expected string
	}{
		{79600, "77.7G"},
		{512, "512M"},
		{1024, "1.0G"},
		{0, "0M"},
	}
	for _, tt := range tests {
		got := FormatMemory(tt.mib)
		if got != tt.expected {
			t.Errorf("FormatMemory(%d) = %q, want %q", tt.mib, got, tt.expected)
		}
	}
}

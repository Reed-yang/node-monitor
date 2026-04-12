package components

import "testing"

func TestComputeDisplayNames_CommonPrefix(t *testing.T) {
	hosts := []string{
		"host-10-240-99-115",
		"host-10-240-99-116",
		"host-10-240-99-117",
		"host-10-240-99-118",
		"host-10-240-99-119",
		"host-10-240-99-120",
	}
	names := ComputeDisplayNames(hosts)
	expected := []string{"115", "116", "117", "118", "119", "120"}
	for i, want := range expected {
		if names[hosts[i]] != want {
			t.Errorf("host %q: got %q, want %q", hosts[i], names[hosts[i]], want)
		}
	}
}

func TestComputeDisplayNames_SingleNode(t *testing.T) {
	hosts := []string{"gpu-server-01"}
	names := ComputeDisplayNames(hosts)
	if names["gpu-server-01"] != "gpu-server-01" {
		t.Errorf("single node: got %q, want full hostname", names["gpu-server-01"])
	}
}

func TestComputeDisplayNames_NothingInCommon(t *testing.T) {
	hosts := []string{"alpha", "beta", "gamma"}
	names := ComputeDisplayNames(hosts)
	if names["alpha"] != "alpha" {
		t.Errorf("no common prefix: got %q, want %q", names["alpha"], "alpha")
	}
}

func TestComputeDisplayNames_PartialPrefix(t *testing.T) {
	hosts := []string{"node-a1", "node-a2", "node-b1"}
	names := ComputeDisplayNames(hosts)
	expected := map[string]string{"node-a1": "a1", "node-a2": "a2", "node-b1": "b1"}
	for h, want := range expected {
		if names[h] != want {
			t.Errorf("host %q: got %q, want %q", h, names[h], want)
		}
	}
}

func TestComputeDisplayNames_EmptyAfterStrip(t *testing.T) {
	hosts := []string{"same", "same"}
	names := ComputeDisplayNames(hosts)
	if names["same"] != "same" {
		t.Errorf("identical hosts: got %q, want full hostname", names["same"])
	}
}

func TestTruncateHostname(t *testing.T) {
	if got := TruncateHostname("very-long-hostname-that-wont-fit", 10); got != "..wont-fit" {
		t.Errorf("got %q, want %q", got, "..wont-fit")
	}
	if got := TruncateHostname("short", 10); got != "short" {
		t.Errorf("got %q, want %q", got, "short")
	}
}

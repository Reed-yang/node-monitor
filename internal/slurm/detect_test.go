package slurm

import "testing"

func TestParseSinfoOutput(t *testing.T) {
	output := "visko-1\nvisko-2\nvisko-1\nvisko-3\n\n"
	nodes := parseSinfoOutput(output)

	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	expected := []string{"visko-1", "visko-2", "visko-3"}
	for i, n := range nodes {
		if n != expected[i] {
			t.Errorf("node[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestParseSinfoOutput_Empty(t *testing.T) {
	nodes := parseSinfoOutput("")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

func TestParseNodeList(t *testing.T) {
	nodes := ParseNodeList("visko-1, visko-2 , visko-3")
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	if nodes[0] != "visko-1" || nodes[2] != "visko-3" {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

func TestParseNodeList_Empty(t *testing.T) {
	nodes := ParseNodeList("")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

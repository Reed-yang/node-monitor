package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Interval != 2.0 {
		t.Errorf("Interval = %f, want 2.0", cfg.Interval)
	}
	if cfg.Workers != 8 {
		t.Errorf("Workers = %d, want 8", cfg.Workers)
	}
	if cfg.View != "panel" {
		t.Errorf("View = %q, want panel", cfg.View)
	}
	if cfg.SSH.ConnectTimeout != 5 {
		t.Errorf("SSH.ConnectTimeout = %d, want 5", cfg.SSH.ConnectTimeout)
	}
	if cfg.SSH.CommandTimeout != 10 {
		t.Errorf("SSH.CommandTimeout = %d, want 10", cfg.SSH.CommandTimeout)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `
nodes = ["node-1", "node-2"]
interval = 5.0
workers = 4
view = "compact"
processes = true

[ssh]
connect_timeout = 10
command_timeout = 20
user = "testuser"

[groups]
train = ["node-1"]
inference = ["node-2"]
`
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	if len(cfg.Nodes) != 2 || cfg.Nodes[0] != "node-1" {
		t.Errorf("Nodes = %v, want [node-1, node-2]", cfg.Nodes)
	}
	if cfg.Interval != 5.0 {
		t.Errorf("Interval = %f, want 5.0", cfg.Interval)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, want 4", cfg.Workers)
	}
	if cfg.View != "compact" {
		t.Errorf("View = %q, want compact", cfg.View)
	}
	if !cfg.Processes {
		t.Error("Processes = false, want true")
	}
	if cfg.SSH.ConnectTimeout != 10 {
		t.Errorf("SSH.ConnectTimeout = %d, want 10", cfg.SSH.ConnectTimeout)
	}
	if cfg.SSH.User != "testuser" {
		t.Errorf("SSH.User = %q, want testuser", cfg.SSH.User)
	}
	if trains, ok := cfg.Groups["train"]; !ok || len(trains) != 1 {
		t.Errorf("Groups[train] = %v, want [node-1]", trains)
	}
}

func TestResolveNodes_GroupOverride(t *testing.T) {
	cfg := Defaults()
	cfg.Nodes = []string{"a", "b"}
	cfg.Groups = map[string][]string{
		"train": {"x", "y"},
	}

	nodes := cfg.ResolveNodes("train")
	if len(nodes) != 2 || nodes[0] != "x" {
		t.Errorf("ResolveNodes(train) = %v, want [x, y]", nodes)
	}

	nodes = cfg.ResolveNodes("")
	if len(nodes) != 2 || nodes[0] != "a" {
		t.Errorf("ResolveNodes('') = %v, want [a, b]", nodes)
	}
}

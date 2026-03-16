package slurm

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// DetectNodes runs sinfo to discover Slurm nodes.
func DetectNodes() ([]string, error) {
	ctx_timeout := 10 * time.Second
	cmd := exec.Command("sinfo", "-h", "-o", "%n")

	done := make(chan error, 1)
	var output []byte
	var cmdErr error

	go func() {
		output, cmdErr = cmd.Output()
		done <- cmdErr
	}()

	select {
	case <-time.After(ctx_timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("sinfo command timed out")
	case err := <-done:
		if err != nil {
			if execErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("sinfo failed: %s", string(execErr.Stderr))
			}
			if _, ok := err.(*exec.Error); ok {
				return nil, fmt.Errorf("slurm is not installed or sinfo is not in PATH")
			}
			return nil, fmt.Errorf("sinfo failed: %w", err)
		}
	}

	nodes := parseSinfoOutput(string(output))
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found from sinfo")
	}
	return nodes, nil
}

func parseSinfoOutput(output string) []string {
	seen := make(map[string]bool)
	var nodes []string
	for _, line := range strings.Split(output, "\n") {
		node := strings.TrimSpace(line)
		if node != "" && !seen[node] {
			seen[node] = true
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	return nodes
}

// ParseNodeList parses a comma-separated node list string.
func ParseNodeList(s string) []string {
	var nodes []string
	for _, n := range strings.Split(s, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

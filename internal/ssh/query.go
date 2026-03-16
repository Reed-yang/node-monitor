package ssh

import (
	"sort"
	"sync"

	"github.com/siyuan/node-monitor/internal/model"
)

// QueryNode queries GPU info from a single node.
func (p *Pool) QueryNode(host string, cmdTimeout int, debug bool) model.NodeStatus {
	output, err := p.RunCommand(host, ListViewCommand(), cmdTimeout)
	if err != nil {
		errMsg := err.Error()
		if !debug {
			if len(errMsg) > 40 {
				errMsg = errMsg[:40]
			}
		}
		return model.NodeStatus{Hostname: host, Error: &errMsg}
	}

	gpus := parseGPUOutput(output)
	return model.NodeStatus{Hostname: host, GPUs: gpus}
}

// QueryNodeDetail queries detailed GPU, process, and system info from a node.
func (p *Pool) QueryNodeDetail(host string, cmdTimeout int, debug bool) (model.NodeStatus, *model.SystemInfo) {
	output, err := p.RunCommand(host, DetailViewCommand(), cmdTimeout)
	if err != nil {
		errMsg := err.Error()
		if !debug {
			if len(errMsg) > 40 {
				errMsg = errMsg[:40]
			}
		}
		return model.NodeStatus{Hostname: host, Error: &errMsg}, nil
	}

	result := parseDetailOutput(output)
	return model.NodeStatus{Hostname: host, GPUs: result.GPUs}, result.System
}

// QueryAllNodes queries all nodes in parallel, limited by maxWorkers.
func (p *Pool) QueryAllNodes(hosts []string, cmdTimeout int, debug bool, maxWorkers ...int) []model.NodeStatus {
	var wg sync.WaitGroup
	results := make([]model.NodeStatus, len(hosts))

	workerLimit := 8
	if len(maxWorkers) > 0 && maxWorkers[0] > 0 {
		workerLimit = maxWorkers[0]
	}
	sem := make(chan struct{}, workerLimit)

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = p.QueryNode(h, cmdTimeout, debug)
		}(i, host)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Hostname < results[j].Hostname
	})

	return results
}

package ssh

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Reed-yang/node-monitor/internal/model"
)

// DetailResult holds the parsed output from a detail query.
type DetailResult struct {
	GPUs   []model.GPUInfo
	System *model.SystemInfo
}

func parseGPUOutput(output string) []model.GPUInfo {
	var gpus []model.GPUInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		idx, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		util, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		memUsed, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
		memTotal, err4 := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		name := ""
		if len(parts) >= 5 {
			name = strings.TrimSpace(parts[4])
		}
		gpus = append(gpus, model.GPUInfo{
			Index:       idx,
			Utilization: util,
			MemoryUsed:  memUsed,
			MemoryTotal: memTotal,
			Name:        name,
		})
	}
	return gpus
}

func parseDetailOutput(output string) DetailResult {
	result := DetailResult{}

	sections := splitSections(output)

	result.GPUs = parseGPUOutput(sections["gpu"])

	// Build UUID-to-index map
	uuidToIdx := make(map[string]int)
	for _, line := range strings.Split(sections["uuid_map"], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 2 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		uuidToIdx[strings.TrimSpace(parts[1])] = idx
	}

	// Build PID-to-user map
	pidToUser := make(map[int]string)
	for _, line := range strings.Split(sections["users"], "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pidToUser[pid] = fields[1]
	}

	// Parse processes and assign to GPUs
	gpuMap := make(map[int]*model.GPUInfo)
	for i := range result.GPUs {
		gpuMap[result.GPUs[i].Index] = &result.GPUs[i]
	}

	for _, line := range strings.Split(sections["processes"], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		gpuUUID := strings.TrimSpace(parts[0])
		pid, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		mem, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		cmdRaw := strings.TrimSpace(parts[3])
		cmd := cmdRaw
		if idx := strings.LastIndex(cmd, "/"); idx >= 0 {
			cmd = cmd[idx+1:]
		}
		if len(cmd) > 20 {
			cmd = cmd[:20]
		}

		gpuIdx, ok := uuidToIdx[gpuUUID]
		if !ok {
			continue
		}
		user := pidToUser[pid]
		if user == "" {
			user = "?"
		}
		if len(user) > 8 {
			user = user[:8]
		}

		if gpu, ok := gpuMap[gpuIdx]; ok {
			gpu.Processes = append(gpu.Processes, model.GPUProcess{
				PID:       pid,
				User:      user,
				GPUIndex:  gpuIdx,
				MemoryMiB: mem,
				Command:   cmd,
			})
		}
	}

	// Parse system info
	result.System = parseSystemInfo(sections["system"])

	return result
}

func splitSections(output string) map[string]string {
	sections := map[string]string{
		"gpu":       "",
		"processes": "",
		"uuid_map":  "",
		"users":     "",
		"system":    "",
	}

	markers := []struct {
		marker string
		key    string
	}{
		{"---PROCESSES---", "processes"},
		{"---GPU_UUID_MAP---", "uuid_map"},
		{"---USERS---", "users"},
		{"---SYSTEM---", "system"},
	}

	remaining := output

	// Extract GPU section (everything before first marker)
	if idx := strings.Index(remaining, "---PROCESSES---"); idx >= 0 {
		sections["gpu"] = remaining[:idx]
		remaining = remaining[idx:]
	} else {
		sections["gpu"] = remaining
		return sections
	}

	// Extract each subsequent section
	for i, m := range markers {
		start := strings.Index(remaining, m.marker)
		if start < 0 {
			continue
		}
		contentStart := start + len(m.marker)

		// Find end: next marker or end of string
		end := len(remaining)
		if i+1 < len(markers) {
			if nextIdx := strings.Index(remaining[contentStart:], markers[i+1].marker); nextIdx >= 0 {
				end = contentStart + nextIdx
			}
		}

		sections[m.key] = strings.TrimSpace(remaining[contentStart:end])
		remaining = remaining[end:]
	}

	return sections
}

func parseSystemInfo(section string) *model.SystemInfo {
	if section == "" {
		return nil
	}

	info := &model.SystemInfo{}
	lines := strings.Split(section, "\n")

	// Line 0: /proc/loadavg -> "1.23 2.34 3.45 2/100 12345"
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		if len(fields) >= 3 {
			info.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
			info.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
			info.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	// Find line starting with "Mem:"
	for _, line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				info.MemTotalBytes, _ = strconv.ParseInt(fields[1], 10, 64)
				info.MemUsedBytes, _ = strconv.ParseInt(fields[2], 10, 64)
			}
			break
		}
	}

	// Last non-empty line: driver version
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if line == "N/A" {
			info.DriverVersion = "N/A"
		} else {
			re := regexp.MustCompile(`\d+\.\d+\.\d+`)
			if m := re.FindString(line); m != "" {
				info.DriverVersion = m
			} else {
				info.DriverVersion = line
			}
		}
		break
	}

	return info
}

// ListViewCommand returns the nvidia-smi command for list view queries (GPU stats only).
func ListViewCommand() string {
	return "nvidia-smi --query-gpu=index,utilization.gpu,memory.used,memory.total,name --format=csv,noheader,nounits"
}

// ListWithProcessesCommand returns the command for list view with process info (no system info).
func ListWithProcessesCommand() string {
	return fmt.Sprintf(`%s && echo '---PROCESSES---' && nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory,process_name --format=csv,noheader,nounits 2>/dev/null || true && echo '---GPU_UUID_MAP---' && nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits 2>/dev/null || true && echo '---USERS---' && ps -eo pid,user --no-headers 2>/dev/null || true`,
		ListViewCommand())
}

// DetailViewCommand returns the batched command for detail view queries.
func DetailViewCommand() string {
	return fmt.Sprintf(`%s && echo '---PROCESSES---' && nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory,process_name --format=csv,noheader,nounits 2>/dev/null || true && echo '---GPU_UUID_MAP---' && nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits 2>/dev/null || true && echo '---USERS---' && ps -eo pid,user --no-headers 2>/dev/null || true && echo '---SYSTEM---' && cat /proc/loadavg && free -b | head -2 && cat /proc/driver/nvidia/version 2>/dev/null || echo 'N/A'`,
		ListViewCommand())
}

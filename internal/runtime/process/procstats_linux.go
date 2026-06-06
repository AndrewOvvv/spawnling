//go:build linux

package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// OSProcStats reads process stats from /proc on Linux.
type OSProcStats struct{}

func (OSProcStats) Read(pid int) (cpuPct float64, memMB int, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.Atoi(fields[1])
				memMB = kb / 1024
			}
			break
		}
	}
	// Accurate CPU% requires two samples; returning 0 is acceptable for now.
	return 0, memMB, nil
}

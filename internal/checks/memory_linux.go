//go:build linux

package checks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkMemory(_ context.Context, cfg config.CheckConfig) Result {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return fail("open /proc/meminfo failed: " + err.Error())
	}
	defer f.Close()

	vals := map[string]uint64{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		vals[key] = v
	}
	if err := s.Err(); err != nil {
		return fail("read /proc/meminfo failed: " + err.Error())
	}

	total := vals["MemTotal"]
	available := vals["MemAvailable"]
	if total == 0 || available == 0 {
		return fail("MemTotal/MemAvailable not found")
	}
	usedPercent := float64(total-available) / float64(total) * 100
	if usedPercent >= cfg.MaxMemoryUsedPercent {
		return fail(fmt.Sprintf("memory usage %.1f%% >= %.1f%%", usedPercent, cfg.MaxMemoryUsedPercent))
	}
	return ok(fmt.Sprintf("memory usage %.1f%%", usedPercent))
}

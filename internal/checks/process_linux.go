//go:build linux

package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkProcess(ctx context.Context, cfg config.CheckConfig) Result {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return fail("failed to read /proc: " + err.Error())
	}
	needle := strings.TrimSpace(cfg.Process)
	if needle == "" {
		return fail("process name is empty")
	}

	matches := 0
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return fail("process check timeout: " + ctx.Err().Error())
		default:
		}
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		base := filepath.Join("/proc", e.Name())
		commBytes, _ := os.ReadFile(filepath.Join(base, "comm"))
		comm := strings.TrimSpace(string(commBytes))
		if comm == needle {
			matches++
			continue
		}
		cmdBytes, _ := os.ReadFile(filepath.Join(base, "cmdline"))
		cmd := strings.ReplaceAll(string(cmdBytes), "\x00", " ")
		if strings.Contains(cmd, needle) {
			matches++
		}
	}

	if matches == 0 {
		return fail(fmt.Sprintf("process %q not found", needle))
	}
	return ok(fmt.Sprintf("process %q found: matches=%d", needle, matches))
}

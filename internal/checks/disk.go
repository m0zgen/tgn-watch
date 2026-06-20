package checks

import (
	"context"
	"fmt"
	"syscall"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkDisk(_ context.Context, cfg config.CheckConfig) Result {
	var st syscall.Statfs_t
	if err := syscall.Statfs(cfg.Path, &st); err != nil {
		return fail("statfs failed: " + err.Error())
	}
	total := float64(st.Blocks) * float64(st.Bsize)
	free := float64(st.Bavail) * float64(st.Bsize)
	if total <= 0 {
		return fail("invalid filesystem size")
	}
	usedPercent := (total - free) / total * 100
	if usedPercent >= cfg.MaxUsedPercent {
		return fail(fmt.Sprintf("disk usage %.1f%% >= %.1f%%", usedPercent, cfg.MaxUsedPercent))
	}
	return ok(fmt.Sprintf("disk usage %.1f%%", usedPercent))
}

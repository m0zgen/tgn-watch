package checks

import (
	"context"
	"os/exec"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkSystemd(ctx context.Context, cfg config.CheckConfig) Result {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", cfg.Service)
	out, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(out))
	if err != nil {
		if status == "" {
			status = err.Error()
		}
		return fail("systemd service is not active: " + status)
	}
	if status != "active" {
		return fail("systemd service is not active: " + status)
	}
	return ok("systemd service active")
}

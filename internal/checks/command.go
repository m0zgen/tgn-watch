package checks

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkCommand(ctx context.Context, cfg config.CheckConfig) Result {
	cmdStr := strings.TrimSpace(cfg.Command)
	if cmdStr == "" {
		return fail("command is empty")
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if len(output) > 500 {
		output = output[:500] + "..."
	}

	exitCode := 0
	if err != nil {
		exitCode = 1
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}
	if ctx.Err() != nil {
		return fail("command timeout: " + ctx.Err().Error())
	}

	if exitCode != cfg.ExpectExitCode {
		if output == "" {
			return fail(fmt.Sprintf("command exit code mismatch: got %d expected %d", exitCode, cfg.ExpectExitCode))
		}
		return fail(fmt.Sprintf("command exit code mismatch: got %d expected %d output=%q", exitCode, cfg.ExpectExitCode, output))
	}
	if output == "" {
		return ok(fmt.Sprintf("command exit code %d", exitCode))
	}
	return ok(fmt.Sprintf("command exit code %d output=%q", exitCode, output))
}

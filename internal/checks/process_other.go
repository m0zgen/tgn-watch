//go:build !linux

package checks

import (
	"context"
	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkProcess(ctx context.Context, cfg config.CheckConfig) Result {
	return fail("process check is currently supported only on Linux")
}

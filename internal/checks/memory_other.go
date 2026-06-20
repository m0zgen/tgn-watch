//go:build !linux

package checks

import (
	"context"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkMemory(_ context.Context, _ config.CheckConfig) Result {
	return fail("memory check is currently supported only on Linux")
}

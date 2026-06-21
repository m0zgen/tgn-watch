package checks

import (
	"fmt"
	"time"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkFileAge(cfg config.CheckConfig) Result {
	info, err := osStat(cfg.Path)
	if err != nil {
		return fail("file stat failed: " + err.Error())
	}
	age := time.Since(info.ModTime())
	if age > cfg.MaxAge.Duration() {
		return fail(fmt.Sprintf("file %s is too old: age=%s max_age=%s", cfg.Path, age.Round(time.Second), cfg.MaxAge.String()))
	}
	return ok(fmt.Sprintf("file %s age=%s", cfg.Path, age.Round(time.Second)))
}

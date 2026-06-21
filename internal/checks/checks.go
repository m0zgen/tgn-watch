package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/m0zgen/tgn-watch/internal/config"
)

type Status string

const (
	StatusOK   Status = "OK"
	StatusFail Status = "FAIL"
)

type Result struct {
	Name     string
	Type     string
	Severity string
	Group    string
	Status   Status
	Message  string
	Duration time.Duration
	Checked  time.Time
}

type Checker interface {
	Check(ctx context.Context, cfg config.CheckConfig) Result
}

func Run(ctx context.Context, cfg config.CheckConfig) Result {
	start := time.Now()
	var res Result

	switch cfg.Type {
	case "http":
		res = checkHTTP(ctx, cfg)
	case "tcp":
		res = checkTCP(ctx, cfg)
	case "systemd":
		res = checkSystemd(ctx, cfg)
	case "disk":
		res = checkDisk(ctx, cfg)
	case "memory":
		res = checkMemory(ctx, cfg)
	case "tls_cert":
		res = checkTLSCert(ctx, cfg)
	case "dns":
		res = checkDNS(ctx, cfg)
	case "process":
		res = checkProcess(ctx, cfg)
	case "file_age":
		res = checkFileAge(cfg)
	case "command":
		res = checkCommand(ctx, cfg)
	default:
		res = Result{Status: StatusFail, Message: fmt.Sprintf("unsupported check type: %s", cfg.Type)}
	}

	res.Name = cfg.Name
	res.Type = cfg.Type
	res.Severity = cfg.Severity
	res.Group = cfg.Group
	res.Duration = time.Since(start)
	res.Checked = time.Now()
	return res
}

func ok(msg string) Result   { return Result{Status: StatusOK, Message: msg} }
func fail(msg string) Result { return Result{Status: StatusFail, Message: msg} }

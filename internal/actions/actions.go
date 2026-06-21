package actions

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	Command  string
	OK       bool
	ExitCode int
	Output   string
	Duration time.Duration
	Error    string
}

func Run(ctx context.Context, command string) Result {
	start := time.Now()
	command = strings.TrimSpace(command)
	res := Result{Command: command, ExitCode: 0}
	if command == "" {
		res.Error = "action command is empty"
		res.ExitCode = -1
		res.Duration = time.Since(start)
		return res
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	out, err := cmd.CombinedOutput()
	res.Output = trimOutput(string(out), 500)
	res.Duration = time.Since(start)

	if ctx.Err() != nil {
		res.ExitCode = -1
		res.Error = ctx.Err().Error()
		return res
	}

	if err != nil {
		res.ExitCode = 1
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		}
		res.Error = err.Error()
		return res
	}

	res.OK = true
	return res
}

func (r Result) Summary() string {
	status := "OK"
	if !r.OK {
		status = "FAIL"
	}
	parts := []string{fmt.Sprintf("action %s exit=%d duration=%s", status, r.ExitCode, r.Duration.Round(time.Millisecond))}
	if r.Output != "" {
		parts = append(parts, fmt.Sprintf("output=%q", r.Output))
	}
	if r.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%q", r.Error))
	}
	return strings.Join(parts, " ")
}

func trimOutput(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

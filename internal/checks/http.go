package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkHTTP(ctx context.Context, cfg config.CheckConfig) Result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return fail("request build failed: " + err.Error())
	}
	req.Header.Set("User-Agent", "tgn-watch/0.1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fail("http request failed: " + err.Error())
	}
	defer resp.Body.Close()

	if cfg.ExpectStatus > 0 && resp.StatusCode != cfg.ExpectStatus {
		return fail(fmt.Sprintf("HTTP status mismatch: got %d, expected %d", resp.StatusCode, cfg.ExpectStatus))
	}
	if cfg.ExpectStatusMin > 0 && resp.StatusCode < cfg.ExpectStatusMin {
		return fail(fmt.Sprintf("HTTP status too low: got %d, expected >= %d", resp.StatusCode, cfg.ExpectStatusMin))
	}
	if cfg.ExpectStatusMax > 0 && resp.StatusCode > cfg.ExpectStatusMax {
		return fail(fmt.Sprintf("HTTP status too high: got %d, expected <= %d", resp.StatusCode, cfg.ExpectStatusMax))
	}
	if cfg.ExpectStatus == 0 && cfg.ExpectStatusMin == 0 && cfg.ExpectStatusMax == 0 && (resp.StatusCode < 200 || resp.StatusCode >= 400) {
		return fail(fmt.Sprintf("HTTP unhealthy status: %d", resp.StatusCode))
	}

	if cfg.ExpectContains != "" {
		b, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		if err != nil {
			return fail("failed to read response body: " + err.Error())
		}
		if !strings.Contains(string(b), cfg.ExpectContains) {
			return fail(fmt.Sprintf("response body does not contain %q", cfg.ExpectContains))
		}
	}

	return ok(fmt.Sprintf("HTTP status %d", resp.StatusCode))
}

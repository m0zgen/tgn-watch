package checks

import (
	"context"
	"net"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkTCP(ctx context.Context, cfg config.CheckConfig) Result {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", cfg.Target)
	if err != nil {
		return fail("tcp connect failed: " + err.Error())
	}
	_ = conn.Close()
	return ok("tcp connect ok")
}

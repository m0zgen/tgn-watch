package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/m0zgen/tgn-watch/internal/config"
)

func checkTLSCert(ctx context.Context, cfg config.CheckConfig) Result {
	host, _, err := net.SplitHostPort(cfg.Target)
	if err != nil {
		return fail("invalid target: " + err.Error())
	}

	d := tls.Dialer{Config: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}
	conn, err := d.DialContext(ctx, "tcp", cfg.Target)
	if err != nil {
		return fail("tls connect failed: " + err.Error())
	}
	defer conn.Close()

	tlsConn, tlsOK := conn.(*tls.Conn)
	if !tlsOK {
		return fail("internal error: not a tls connection")
	}
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return fail("no peer certificates")
	}
	cert := state.PeerCertificates[0]
	daysLeft := time.Until(cert.NotAfter).Hours() / 24

	if daysLeft <= float64(cfg.CriticalDays) {
		return fail(fmt.Sprintf("TLS certificate expires in %.1f days, critical threshold %d days", daysLeft, cfg.CriticalDays))
	}
	if daysLeft <= float64(cfg.WarnDays) {
		return fail(fmt.Sprintf("TLS certificate expires in %.1f days, warning threshold %d days", daysLeft, cfg.WarnDays))
	}
	return ok(fmt.Sprintf("TLS certificate valid for %.1f days", daysLeft))
}

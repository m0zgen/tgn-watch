package checks

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/m0zgen/tgn-watch/internal/config"
)

var dnsQTypes = map[string]uint16{
	"A":     1,
	"NS":    2,
	"CNAME": 5,
	"SOA":   6,
	"PTR":   12,
	"MX":    15,
	"TXT":   16,
	"AAAA":  28,
	"SRV":   33,
}

var dnsRCodes = map[int]string{
	0: "NOERROR",
	1: "FORMERR",
	2: "SERVFAIL",
	3: "NXDOMAIN",
	4: "NOTIMP",
	5: "REFUSED",
}

func checkDNS(ctx context.Context, cfg config.CheckConfig) Result {
	qtype, found := dnsQTypes[strings.ToUpper(cfg.QType)]
	if !found {
		return fail(fmt.Sprintf("unsupported qtype %q", cfg.QType))
	}

	query, id, err := buildDNSQuery(cfg.QName, qtype)
	if err != nil {
		return fail("failed to build DNS query: " + err.Error())
	}

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "udp", cfg.Server)
	if err != nil {
		return fail("DNS dial failed: " + err.Error())
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if _, err := conn.Write(query); err != nil {
		return fail("DNS write failed: " + err.Error())
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return fail("DNS read failed: " + err.Error())
	}
	if n < 12 {
		return fail(fmt.Sprintf("DNS response too short: %d bytes", n))
	}
	if gotID := binary.BigEndian.Uint16(buf[0:2]); gotID != id {
		return fail(fmt.Sprintf("DNS transaction id mismatch: got %d expected %d", gotID, id))
	}

	rcodeNum := int(buf[3] & 0x0f)
	rcode := dnsRCodes[rcodeNum]
	if rcode == "" {
		rcode = fmt.Sprintf("RCODE%d", rcodeNum)
	}
	if want := strings.ToUpper(cfg.ExpectRCode); want != "" && rcode != want {
		return fail(fmt.Sprintf("DNS rcode mismatch: got %s expected %s", rcode, want))
	}

	answers := int(binary.BigEndian.Uint16(buf[6:8]))
	if cfg.ExpectMinAnswers > 0 && answers < cfg.ExpectMinAnswers {
		return fail(fmt.Sprintf("DNS answer count too low: got %d expected >= %d", answers, cfg.ExpectMinAnswers))
	}

	return ok(fmt.Sprintf("DNS %s %s rcode=%s answers=%d", strings.ToUpper(cfg.QType), cfg.QName, rcode, answers))
}

func buildDNSQuery(qname string, qtype uint16) ([]byte, uint16, error) {
	var idBuf [2]byte
	if _, err := rand.Read(idBuf[:]); err != nil {
		return nil, 0, err
	}
	id := binary.BigEndian.Uint16(idBuf[:])

	msg := make([]byte, 0, 512)
	msg = append(msg, idBuf[:]...)
	msg = append(msg, 0x01, 0x00) // standard query, recursion desired
	msg = append(msg, 0x00, 0x01) // QDCOUNT
	msg = append(msg, 0x00, 0x00) // ANCOUNT
	msg = append(msg, 0x00, 0x00) // NSCOUNT
	msg = append(msg, 0x00, 0x00) // ARCOUNT

	name := strings.TrimSuffix(qname, ".")
	if name == "" {
		msg = append(msg, 0)
	} else {
		for _, label := range strings.Split(name, ".") {
			if len(label) == 0 || len(label) > 63 {
				return nil, 0, fmt.Errorf("invalid DNS label %q", label)
			}
			msg = append(msg, byte(len(label)))
			msg = append(msg, label...)
		}
		msg = append(msg, 0)
	}
	msg = binary.BigEndian.AppendUint16(msg, qtype)
	msg = binary.BigEndian.AppendUint16(msg, 1) // IN
	return msg, id, nil
}

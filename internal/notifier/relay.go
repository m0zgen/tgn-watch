package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/m0zgen/tgn-watch/internal/checks"
	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/state"
)

type Relay struct {
	endpoint string
	key      string
	client   *http.Client
}

type payload struct {
	Group     string `json:"group"`
	ParseMode string `json:"parse_mode"`
	Text      string `json:"text"`
}

func NewRelay(cfg config.RelayConfig) *Relay {
	return &Relay{
		endpoint: cfg.Endpoint,
		key:      cfg.Key,
		client:   &http.Client{Timeout: cfg.Timeout.Duration()},
	}
}

func (r *Relay) Notify(ctx context.Context, res checks.Result, tr state.Transition, host string) error {
	p := payload{Group: res.Group, ParseMode: "HTML", Text: renderHTML(res, tr, host)}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Relay-Key", r.key)
	req.Header.Set("User-Agent", "tgn-watch/0.1.2")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("relay returned status %d", resp.StatusCode)
	}
	return nil
}

func renderHTML(res checks.Result, tr state.Transition, host string) string {
	icon := "🚨"
	title := "Check failed"
	status := string(res.Status)
	if tr == state.TransitionRecovery {
		icon = "✅"
		title = "Check restored"
		status = "OK"
	} else if tr == state.TransitionRepeat {
		icon = "🔁"
		title = "Check still failing"
	}

	sev := strings.ToUpper(res.Severity)
	return fmt.Sprintf(`<b>%s %s — %s</b>

<b>Host:</b> <code>%s</code>
<b>Check:</b> <code>%s</code>
<b>Type:</b> <code>%s</code>
<b>Status:</b> <code>%s</code>
<b>Duration:</b> <code>%s</code>

<blockquote>%s</blockquote>`,
		icon,
		html.EscapeString(sev),
		html.EscapeString(title),
		html.EscapeString(host),
		html.EscapeString(res.Name),
		html.EscapeString(res.Type),
		html.EscapeString(status),
		html.EscapeString(res.Duration.Round(time.Millisecond).String()),
		html.EscapeString(res.Message),
	)
}

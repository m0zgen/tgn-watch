package runner

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/m0zgen/tgn-watch/internal/checks"
	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/notifier"
	"github.com/m0zgen/tgn-watch/internal/state"
)

type Runner struct {
	cfg      *config.Config
	notifier *notifier.Relay
	state    *state.Store
	lastRun  map[string]time.Time
	mu       sync.Mutex
}

func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg:      cfg,
		notifier: notifier.NewRelay(cfg.Relay),
		state:    state.New(),
		lastRun:  make(map[string]time.Time),
	}
}

func (r *Runner) Run(ctx context.Context) error {
	log.Printf("tgn-watch started: checks=%d interval=%s host=%s", len(r.cfg.Checks), r.cfg.Watcher.Interval.String(), r.cfg.Watcher.Hostname)
	r.runOnce(ctx)

	ticker := time.NewTicker(r.cfg.Watcher.Interval.Duration())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	var wg sync.WaitGroup
	for _, ch := range r.cfg.Checks {
		ch := ch
		if !r.shouldRun(ch) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runCheck(ctx, ch)
		}()
	}
	wg.Wait()
}

func (r *Runner) shouldRun(ch config.CheckConfig) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	last := r.lastRun[ch.Name]
	if !last.IsZero() && now.Sub(last) < ch.Interval.Duration() {
		return false
	}
	r.lastRun[ch.Name] = now
	return true
}

func (r *Runner) runCheck(parent context.Context, ch config.CheckConfig) {
	ctx, cancel := context.WithTimeout(parent, ch.Timeout.Duration())
	defer cancel()

	res := checks.Run(ctx, ch)
	tr := r.state.Update(ch.Name, res, r.cfg.Watcher.DedupWindow.Duration(), r.cfg.Watcher.NotifyOnRecovery)

	if res.Status == checks.StatusOK {
		log.Printf("check=%q type=%s status=OK msg=%q duration=%s", res.Name, res.Type, res.Message, res.Duration.Round(time.Millisecond))
	} else {
		log.Printf("check=%q type=%s status=FAIL msg=%q duration=%s transition=%s", res.Name, res.Type, res.Message, res.Duration.Round(time.Millisecond), tr)
	}

	if tr == state.TransitionNone {
		return
	}

	nctx, cancel := context.WithTimeout(parent, r.cfg.Relay.Timeout.Duration())
	defer cancel()
	if err := r.notifier.Notify(nctx, res, tr, r.cfg.Watcher.Hostname); err != nil {
		log.Printf("notify failed: check=%q error=%v", res.Name, err)
	}
}

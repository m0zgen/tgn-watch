package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/m0zgen/tgn-watch/internal/actions"
	"github.com/m0zgen/tgn-watch/internal/checks"
	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/notifier"
	"github.com/m0zgen/tgn-watch/internal/state"
)

type Runner struct {
	cfg        *config.Config
	notifier   *notifier.Relay
	state      *state.Store
	lastRun    map[string]time.Time
	lastAction map[string]time.Time
	mu         sync.Mutex
}

func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg:        cfg,
		notifier:   notifier.NewRelay(cfg.Relay),
		state:      state.New(),
		lastRun:    make(map[string]time.Time),
		lastAction: make(map[string]time.Time),
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

func (r *Runner) canRunAction(ch config.CheckConfig) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	last := r.lastAction[ch.Name]
	if !last.IsZero() {
		remaining := ch.ActionCooldown.Duration() - now.Sub(last)
		if remaining > 0 {
			return false, remaining
		}
	}
	r.lastAction[ch.Name] = now
	return true, 0
}

func (r *Runner) runCheck(parent context.Context, ch config.CheckConfig) {
	ctx, cancel := context.WithTimeout(parent, ch.Timeout.Duration())
	res := checks.Run(ctx, ch)
	cancel()

	if res.Status == checks.StatusFail && ch.ActionEnabled {
		res = r.tryAutoAction(parent, ch, res)
	}

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

func (r *Runner) tryAutoAction(parent context.Context, ch config.CheckConfig, initial checks.Result) checks.Result {
	allowed, remaining := r.canRunAction(ch)
	if !allowed {
		initial.Message = fmt.Sprintf("%s; auto action skipped: cooldown remaining %s", initial.Message, remaining.Round(time.Second))
		return initial
	}

	log.Printf("auto_action start: check=%q retries=%d command=%q", ch.Name, ch.ActionRetries, ch.ActionCommand)
	lastActionSummary := ""
	lastCheck := initial

	for attempt := 1; attempt <= ch.ActionRetries; attempt++ {
		actCtx, cancel := context.WithTimeout(parent, ch.ActionTimeout.Duration())
		act := actions.Run(actCtx, ch.ActionCommand)
		cancel()
		lastActionSummary = act.Summary()
		log.Printf("auto_action attempt=%d check=%q %s", attempt, ch.Name, lastActionSummary)

		if !sleepContext(parent, ch.ActionDelay.Duration()) {
			lastCheck.Message = fmt.Sprintf("%s; auto action interrupted after attempt %d: %s", initial.Message, attempt, lastActionSummary)
			return lastCheck
		}

		reCtx, cancel := context.WithTimeout(parent, ch.Timeout.Duration())
		recheck := checks.Run(reCtx, ch)
		cancel()
		lastCheck = recheck
		log.Printf("auto_action recheck attempt=%d check=%q status=%s msg=%q", attempt, ch.Name, recheck.Status, recheck.Message)

		if recheck.Status == checks.StatusOK {
			recheck.Message = fmt.Sprintf("auto action recovered after attempt %d; %s; recheck: %s", attempt, lastActionSummary, recheck.Message)
			return recheck
		}
	}

	lastCheck.Message = fmt.Sprintf("%s; auto action failed after %d attempt(s); last action: %s; last recheck: %s", initial.Message, ch.ActionRetries, lastActionSummary, lastCheck.Message)
	return lastCheck
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

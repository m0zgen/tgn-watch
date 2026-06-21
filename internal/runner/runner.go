package runner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/m0zgen/tgn-watch/internal/actions"
	"github.com/m0zgen/tgn-watch/internal/checks"
	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/notifier"
	"github.com/m0zgen/tgn-watch/internal/state"
	"github.com/m0zgen/tgn-watch/internal/status"
	"github.com/m0zgen/tgn-watch/internal/version"
)

type Runner struct {
	cfg        *config.Config
	notifier   *notifier.Relay
	state      *state.Store
	lastRun    map[string]time.Time
	lastAction map[string]time.Time
	mu         sync.Mutex
	startedAt  time.Time
	lastRunAt  atomic.Int64
	metrics    Metrics
}

type Metrics struct {
	ChecksRunTotal          atomic.Uint64
	ChecksOKTotal           atomic.Uint64
	ChecksFailTotal         atomic.Uint64
	ActionsAttemptTotal     atomic.Uint64
	ActionsRecoveredTotal   atomic.Uint64
	ActionsFailedTotal      atomic.Uint64
	NotificationsTotal      atomic.Uint64
	NotificationErrorsTotal atomic.Uint64
}

func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg:        cfg,
		notifier:   notifier.NewRelay(cfg.Relay),
		state:      state.New(),
		lastRun:    make(map[string]time.Time),
		lastAction: make(map[string]time.Time),
		startedAt:  time.Now(),
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
	r.lastRunAt.Store(time.Now().Unix())
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

	r.metrics.ChecksRunTotal.Add(1)
	if res.Status == checks.StatusOK {
		r.metrics.ChecksOKTotal.Add(1)
	} else {
		r.metrics.ChecksFailTotal.Add(1)
	}

	if res.Status == checks.StatusFail && ch.ActionEnabled {
		if r.cfg.Watcher.ActionsEnabled {
			res = r.tryAutoAction(parent, ch, res)
		} else {
			log.Printf("auto_action skipped: check=%q reason=global_actions_disabled", ch.Name)
		}
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

	r.metrics.NotificationsTotal.Add(1)
	nctx, cancel := context.WithTimeout(parent, r.cfg.Relay.Timeout.Duration())
	defer cancel()
	if err := r.notifier.Notify(nctx, res, tr, r.cfg.Watcher.Hostname); err != nil {
		r.metrics.NotificationErrorsTotal.Add(1)
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
		r.metrics.ActionsAttemptTotal.Add(1)
		actCtx, cancel := context.WithTimeout(parent, ch.ActionTimeout.Duration())
		act := actions.Run(actCtx, ch.ActionCommand)
		cancel()
		lastActionSummary = act.Summary()
		log.Printf("auto_action attempt=%d check=%q %s", attempt, ch.Name, lastActionSummary)

		if !sleepContext(parent, ch.ActionDelay.Duration()) {
			lastCheck.Message = fmt.Sprintf("%s; auto action interrupted after attempt %d: %s", initial.Message, attempt, lastActionSummary)
			r.metrics.ActionsFailedTotal.Add(1)
			return lastCheck
		}

		reCtx, cancel := context.WithTimeout(parent, ch.Timeout.Duration())
		recheck := checks.Run(reCtx, ch)
		cancel()
		lastCheck = recheck
		log.Printf("auto_action recheck attempt=%d check=%q status=%s msg=%q", attempt, ch.Name, recheck.Status, recheck.Message)

		if recheck.Status == checks.StatusOK {
			recheck.Message = fmt.Sprintf("auto action recovered after attempt %d; %s; recheck: %s", attempt, lastActionSummary, recheck.Message)
			r.metrics.ActionsRecoveredTotal.Add(1)
			return recheck
		}
	}

	r.metrics.ActionsFailedTotal.Add(1)
	lastCheck.Message = fmt.Sprintf("%s; auto action failed after %d attempt(s); last action: %s; last recheck: %s", initial.Message, ch.ActionRetries, lastActionSummary, lastCheck.Message)
	return lastCheck
}

func (r *Runner) Status() status.Runtime {
	checksSnapshot := r.state.Snapshot()
	checksOK := 0
	checksFail := 0
	for _, ch := range checksSnapshot {
		if ch.Status == string(checks.StatusOK) {
			checksOK++
		} else if ch.Status == string(checks.StatusFail) {
			checksFail++
		}
	}

	lastRun := time.Unix(r.lastRunAt.Load(), 0)
	if r.lastRunAt.Load() == 0 {
		lastRun = time.Time{}
	}
	up := time.Since(r.startedAt).Round(time.Second)

	return status.Runtime{
		App:             "tgn-watch",
		Version:         version.Version,
		Commit:          version.Commit,
		Date:            version.Date,
		Host:            r.cfg.Watcher.Hostname,
		StartedAt:       r.startedAt,
		Uptime:          up.String(),
		UptimeSeconds:   int64(up.Seconds()),
		ChecksTotal:     len(r.cfg.Checks),
		ChecksOK:        checksOK,
		ChecksFail:      checksFail,
		WatcherInterval: r.cfg.Watcher.Interval.String(),
		LastRun:         lastRun,
		Metrics: status.Metrics{
			ChecksRunTotal:          r.metrics.ChecksRunTotal.Load(),
			ChecksOKTotal:           r.metrics.ChecksOKTotal.Load(),
			ChecksFailTotal:         r.metrics.ChecksFailTotal.Load(),
			ActionsAttemptTotal:     r.metrics.ActionsAttemptTotal.Load(),
			ActionsRecoveredTotal:   r.metrics.ActionsRecoveredTotal.Load(),
			ActionsFailedTotal:      r.metrics.ActionsFailedTotal.Load(),
			NotificationsTotal:      r.metrics.NotificationsTotal.Load(),
			NotificationErrorsTotal: r.metrics.NotificationErrorsTotal.Load(),
		},
		Checks: checksSnapshot,
	}
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

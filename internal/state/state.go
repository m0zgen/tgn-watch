package state

import (
	"sync"
	"time"

	"github.com/m0zgen/tgn-watch/internal/checks"
	"github.com/m0zgen/tgn-watch/internal/status"
)

type Transition string

const (
	TransitionNone     Transition = "none"
	TransitionAlert    Transition = "alert"
	TransitionRecovery Transition = "recovery"
	TransitionRepeat   Transition = "repeat"
)

type Entry struct {
	Name         string
	Type         string
	Severity     string
	Group        string
	LastStatus   checks.Status
	LastMessage  string
	LastNotify   time.Time
	LastChange   time.Time
	LastCheck    time.Time
	LastDuration time.Duration
}

type Store struct {
	mu    sync.RWMutex
	items map[string]Entry
}

func New() *Store { return &Store{items: make(map[string]Entry)} }

func (s *Store) Update(name string, res checks.Result, dedup time.Duration, notifyRecovery bool) Transition {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.items[name]
	if !exists {
		entry = Entry{
			Name:         name,
			Type:         res.Type,
			Severity:     res.Severity,
			Group:        res.Group,
			LastStatus:   res.Status,
			LastMessage:  res.Message,
			LastChange:   now,
			LastCheck:    res.Checked,
			LastDuration: res.Duration,
		}
		if entry.LastCheck.IsZero() {
			entry.LastCheck = now
		}
		if res.Status == checks.StatusFail {
			entry.LastNotify = now
			s.items[name] = entry
			return TransitionAlert
		}
		s.items[name] = entry
		return TransitionNone
	}

	transition := TransitionNone
	if entry.LastStatus != res.Status {
		entry.LastStatus = res.Status
		entry.LastChange = now
		entry.LastNotify = now
		if res.Status == checks.StatusFail {
			transition = TransitionAlert
		} else if notifyRecovery {
			transition = TransitionRecovery
		}
	} else if res.Status == checks.StatusFail && dedup > 0 && time.Since(entry.LastNotify) >= dedup {
		entry.LastNotify = now
		transition = TransitionRepeat
	}

	entry.Name = name
	entry.Type = res.Type
	entry.Severity = res.Severity
	entry.Group = res.Group
	entry.LastMessage = res.Message
	entry.LastCheck = res.Checked
	if entry.LastCheck.IsZero() {
		entry.LastCheck = now
	}
	entry.LastDuration = res.Duration
	s.items[name] = entry
	return transition
}

func (s *Store) Snapshot() []status.CheckStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]status.CheckStatus, 0, len(s.items))
	for _, e := range s.items {
		out = append(out, status.CheckStatus{
			Name:         e.Name,
			Type:         e.Type,
			Severity:     e.Severity,
			Group:        e.Group,
			Status:       string(e.LastStatus),
			Message:      e.LastMessage,
			LastChanged:  e.LastChange,
			LastChecked:  e.LastCheck,
			LastNotified: e.LastNotify,
			DurationMS:   e.LastDuration.Milliseconds(),
		})
	}
	return out
}

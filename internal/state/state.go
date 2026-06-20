package state

import (
	"time"

	"github.com/m0zgen/tgn-watch/internal/checks"
)

type Transition string

const (
	TransitionNone     Transition = "none"
	TransitionAlert    Transition = "alert"
	TransitionRecovery Transition = "recovery"
	TransitionRepeat   Transition = "repeat"
)

type Entry struct {
	LastStatus checks.Status
	LastNotify time.Time
	LastChange time.Time
}

type Store struct {
	items map[string]Entry
}

func New() *Store { return &Store{items: make(map[string]Entry)} }

func (s *Store) Update(name string, res checks.Result, dedup time.Duration, notifyRecovery bool) Transition {
	now := time.Now()
	entry, exists := s.items[name]
	if !exists {
		s.items[name] = Entry{LastStatus: res.Status, LastChange: now}
		if res.Status == checks.StatusFail {
			entry = s.items[name]
			entry.LastNotify = now
			s.items[name] = entry
			return TransitionAlert
		}
		return TransitionNone
	}

	if entry.LastStatus != res.Status {
		entry.LastStatus = res.Status
		entry.LastChange = now
		entry.LastNotify = now
		s.items[name] = entry
		if res.Status == checks.StatusFail {
			return TransitionAlert
		}
		if notifyRecovery {
			return TransitionRecovery
		}
		return TransitionNone
	}

	if res.Status == checks.StatusFail && dedup > 0 && time.Since(entry.LastNotify) >= dedup {
		entry.LastNotify = now
		s.items[name] = entry
		return TransitionRepeat
	}

	s.items[name] = entry
	return TransitionNone
}

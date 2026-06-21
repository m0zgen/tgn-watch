package status

import "time"

type Runtime struct {
	App             string        `json:"app"`
	Version         string        `json:"version"`
	Commit          string        `json:"commit"`
	Date            string        `json:"date"`
	Host            string        `json:"host"`
	StartedAt       time.Time     `json:"started_at"`
	Uptime          string        `json:"uptime"`
	UptimeSeconds   int64         `json:"uptime_seconds"`
	ChecksTotal     int           `json:"checks_total"`
	ChecksOK        int           `json:"checks_ok"`
	ChecksFail      int           `json:"checks_fail"`
	WatcherInterval string        `json:"watcher_interval"`
	LastRun         time.Time     `json:"last_run"`
	Metrics         Metrics       `json:"metrics"`
	Checks          []CheckStatus `json:"checks"`
}

type Metrics struct {
	ChecksRunTotal          uint64 `json:"checks_run_total"`
	ChecksOKTotal           uint64 `json:"checks_ok_total"`
	ChecksFailTotal         uint64 `json:"checks_fail_total"`
	ActionsAttemptTotal     uint64 `json:"actions_attempt_total"`
	ActionsRecoveredTotal   uint64 `json:"actions_recovered_total"`
	ActionsFailedTotal      uint64 `json:"actions_failed_total"`
	NotificationsTotal      uint64 `json:"notifications_total"`
	NotificationErrorsTotal uint64 `json:"notification_errors_total"`
}

type CheckStatus struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Group        string    `json:"group"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	LastChanged  time.Time `json:"last_changed"`
	LastChecked  time.Time `json:"last_checked"`
	LastNotified time.Time `json:"last_notified"`
	DurationMS   int64     `json:"duration_ms"`
}

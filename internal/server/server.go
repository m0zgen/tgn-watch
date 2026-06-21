package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/status"
)

type Provider interface {
	Status() status.Runtime
}

type Server struct {
	cfg      config.ServerConfig
	provider Provider
}

func New(cfg config.ServerConfig, provider Provider) *Server {
	return &Server{cfg: cfg, provider: provider}
}

func (s *Server) Run(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/metrics", s.handleMetrics)

	srv := &http.Server{
		Addr:              s.cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("status server started: listen=%s", s.cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.provider.Status()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.provider.Status()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	writeMetric(w, "tgn_watch_up", "gauge", "Whether tgn-watch status server is up", 1, nil)
	writeMetric(w, "tgn_watch_uptime_seconds", "gauge", "tgn-watch process uptime in seconds", st.UptimeSeconds, nil)
	writeMetric(w, "tgn_watch_checks_total", "gauge", "Configured checks count", st.ChecksTotal, nil)
	writeMetric(w, "tgn_watch_checks_ok", "gauge", "Checks currently OK", st.ChecksOK, nil)
	writeMetric(w, "tgn_watch_checks_fail", "gauge", "Checks currently failing", st.ChecksFail, nil)

	writeMetric(w, "tgn_watch_checks_run_total", "counter", "Total executed checks", st.Metrics.ChecksRunTotal, nil)
	writeMetric(w, "tgn_watch_checks_ok_total", "counter", "Total successful check executions", st.Metrics.ChecksOKTotal, nil)
	writeMetric(w, "tgn_watch_checks_fail_total", "counter", "Total failed check executions", st.Metrics.ChecksFailTotal, nil)
	writeMetric(w, "tgn_watch_actions_attempt_total", "counter", "Total auto action attempts", st.Metrics.ActionsAttemptTotal, nil)
	writeMetric(w, "tgn_watch_actions_recovered_total", "counter", "Total auto action recoveries", st.Metrics.ActionsRecoveredTotal, nil)
	writeMetric(w, "tgn_watch_actions_failed_total", "counter", "Total failed auto action sequences", st.Metrics.ActionsFailedTotal, nil)
	writeMetric(w, "tgn_watch_notifications_total", "counter", "Total notification attempts", st.Metrics.NotificationsTotal, nil)
	writeMetric(w, "tgn_watch_notification_errors_total", "counter", "Total notification errors", st.Metrics.NotificationErrorsTotal, nil)

	writeMetricHeader(w, "tgn_watch_check_status", "gauge", "Current check status: 1 OK, 0 FAIL")
	writeMetricHeader(w, "tgn_watch_check_last_duration_ms", "gauge", "Last check duration in milliseconds")
	writeMetricHeader(w, "tgn_watch_check_last_checked_timestamp", "gauge", "Unix timestamp of last check execution")
	for _, ch := range st.Checks {
		v := 0
		if ch.Status == "OK" {
			v = 1
		}
		labels := map[string]string{"name": ch.Name, "type": ch.Type, "severity": ch.Severity, "group": ch.Group}
		writeMetricValue(w, "tgn_watch_check_status", v, labels)
		writeMetricValue(w, "tgn_watch_check_last_duration_ms", ch.DurationMS, labels)
		if !ch.LastChecked.IsZero() {
			writeMetricValue(w, "tgn_watch_check_last_checked_timestamp", ch.LastChecked.Unix(), labels)
		}
	}
}

func writeMetric(w http.ResponseWriter, name, typ, help string, value any, labels map[string]string) {
	writeMetricHeader(w, name, typ, help)
	writeMetricValue(w, name, value, labels)
}

func writeMetricHeader(w http.ResponseWriter, name, typ, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, typ)
}

func writeMetricValue(w http.ResponseWriter, name string, value any, labels map[string]string) {
	if len(labels) == 0 {
		fmt.Fprintf(w, "%s %v\n", name, value)
		return
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(labels))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, escapeLabel(labels[k])))
	}
	fmt.Fprintf(w, "%s{%s} %v\n", name, strings.Join(parts, ","), value)
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

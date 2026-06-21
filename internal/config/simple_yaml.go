package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseSimpleYAML(b []byte) (*Config, error) {
	cfg := &Config{}
	section := ""
	var current *CheckConfig

	lines := strings.Split(string(b), "\n")
	for idx, raw := range lines {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := countIndent(line)
		trim := strings.TrimSpace(line)

		if indent == 0 && strings.HasSuffix(trim, ":") && !strings.HasPrefix(trim, "-") {
			section = strings.TrimSuffix(trim, ":")
			current = nil
			continue
		}

		if section == "checks" && indent == 2 && strings.HasPrefix(trim, "- ") {
			cfg.Checks = append(cfg.Checks, CheckConfig{})
			current = &cfg.Checks[len(cfg.Checks)-1]
			kv := strings.TrimSpace(strings.TrimPrefix(trim, "- "))
			if kv != "" {
				key, val, ok := splitKV(kv)
				if !ok {
					return nil, fmt.Errorf("line %d: invalid check item", idx+1)
				}
				if err := setCheckField(current, key, val); err != nil {
					return nil, fmt.Errorf("line %d: %w", idx+1, err)
				}
			}
			continue
		}

		key, val, ok := splitKV(trim)
		if !ok {
			return nil, fmt.Errorf("line %d: invalid key/value", idx+1)
		}

		switch section {
		case "relay":
			if err := setRelayField(&cfg.Relay, key, val); err != nil {
				return nil, fmt.Errorf("line %d: %w", idx+1, err)
			}
		case "watcher":
			if err := setWatcherField(&cfg.Watcher, key, val); err != nil {
				return nil, fmt.Errorf("line %d: %w", idx+1, err)
			}
		case "checks":
			if current == nil {
				return nil, fmt.Errorf("line %d: check field without check item", idx+1)
			}
			if err := setCheckField(current, key, val); err != nil {
				return nil, fmt.Errorf("line %d: %w", idx+1, err)
			}
		default:
			return nil, fmt.Errorf("line %d: unknown section %q", idx+1, section)
		}
	}
	return cfg, nil
}

func stripComment(s string) string {
	inSingle, inDouble := false, false
	for i, r := range s {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return s[:i]
			}
		}
	}
	return s
}

func countIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

func splitKV(s string) (string, string, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), cleanValue(parts[1]), true
}

func cleanValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			unq, err := strconv.Unquote(s)
			if err == nil {
				return unq
			}
			return s[1 : len(s)-1]
		}
	}
	return s
}

func parseDurationValue(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}

func parseBoolValue(s string) (bool, error)     { return strconv.ParseBool(s) }
func parseIntValue(s string) (int, error)       { return strconv.Atoi(s) }
func parseFloatValue(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

func setRelayField(r *RelayConfig, key, val string) error {
	switch key {
	case "endpoint":
		r.Endpoint = val
	case "key":
		r.Key = val
	case "default_group":
		r.DefaultGroup = val
	case "timeout":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		r.Timeout = d
	default:
		return fmt.Errorf("unknown relay field %q", key)
	}
	return nil
}

func setWatcherField(w *WatcherConfig, key, val string) error {
	switch key {
	case "interval":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		w.Interval = d
	case "notify_on_recovery":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.NotifyOnRecovery = b
	case "dedup_window":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		w.DedupWindow = d
	case "hostname":
		w.Hostname = val
	case "command_checks_enabled":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.CommandChecksEnabled = b
	case "actions_enabled":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.ActionsEnabled = b
	default:
		return fmt.Errorf("unknown watcher field %q", key)
	}
	return nil
}

func setCheckField(c *CheckConfig, key, val string) error {
	switch key {
	case "name":
		c.Name = val
	case "type":
		c.Type = val
	case "timeout":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.Timeout = d
	case "group":
		c.Group = val
	case "severity":
		c.Severity = val
	case "interval":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.Interval = d
	case "url":
		c.URL = val
	case "expect_status":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ExpectStatus = i
	case "expect_status_min":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ExpectStatusMin = i
	case "expect_status_max":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ExpectStatusMax = i
	case "expect_contains":
		c.ExpectContains = val
	case "target":
		c.Target = val
	case "service":
		c.Service = val
	case "path":
		c.Path = val
	case "max_used_percent":
		f, err := parseFloatValue(val)
		if err != nil {
			return err
		}
		c.MaxUsedPercent = f
		c.MaxMemoryUsedPercent = f
	case "warn_days":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.WarnDays = i
	case "critical_days":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.CriticalDays = i
	case "server":
		c.Server = val
	case "qname":
		c.QName = val
	case "qtype":
		c.QType = val
	case "expect_rcode":
		c.ExpectRCode = val
	case "expect_min_answers":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ExpectMinAnswers = i
	case "process":
		c.Process = val
	case "max_age":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.MaxAge = d
	case "command":
		c.Command = val
	case "expect_exit_code":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ExpectExitCode = i
	case "action_enabled":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		c.ActionEnabled = b
	case "action_command":
		c.ActionCommand = val
	case "action_retries":
		i, err := parseIntValue(val)
		if err != nil {
			return err
		}
		c.ActionRetries = i
	case "action_timeout":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.ActionTimeout = d
	case "action_delay":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.ActionDelay = d
	case "action_cooldown":
		d, err := parseDurationValue(val)
		if err != nil {
			return err
		}
		c.ActionCooldown = d
	default:
		return fmt.Errorf("unknown check field %q", key)
	}
	return nil
}

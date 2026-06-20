package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

type Config struct {
	Relay   RelayConfig   `yaml:"relay"`
	Watcher WatcherConfig `yaml:"watcher"`
	Checks  []CheckConfig `yaml:"checks"`
}

type RelayConfig struct {
	Endpoint     string   `yaml:"endpoint"`
	Key          string   `yaml:"key"`
	DefaultGroup string   `yaml:"default_group"`
	Timeout      Duration `yaml:"timeout"`
}

type WatcherConfig struct {
	Interval         Duration `yaml:"interval"`
	NotifyOnRecovery bool     `yaml:"notify_on_recovery"`
	DedupWindow      Duration `yaml:"dedup_window"`
	Hostname         string   `yaml:"hostname"`
}

type CheckConfig struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Timeout  Duration `yaml:"timeout"`
	Group    string   `yaml:"group"`
	Severity string   `yaml:"severity"`
	Interval Duration `yaml:"interval"`

	// http
	URL             string `yaml:"url"`
	ExpectStatus    int    `yaml:"expect_status"`
	ExpectStatusMin int    `yaml:"expect_status_min"`
	ExpectStatusMax int    `yaml:"expect_status_max"`
	ExpectContains  string `yaml:"expect_contains"`

	// tcp / tls_cert
	Target string `yaml:"target"`

	// systemd
	Service string `yaml:"service"`

	// disk
	Path           string  `yaml:"path"`
	MaxUsedPercent float64 `yaml:"max_used_percent"`

	// memory
	MaxMemoryUsedPercent float64 `yaml:"max_used_percent,omitempty"`

	// tls_cert
	WarnDays     int `yaml:"warn_days"`
	CriticalDays int `yaml:"critical_days"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := parseSimpleYAML(b)
	if err != nil {
		return nil, err
	}

	applyDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Relay.Timeout == 0 {
		cfg.Relay.Timeout = Duration(3 * time.Second)
	}
	if cfg.Watcher.Interval == 0 {
		cfg.Watcher.Interval = Duration(30 * time.Second)
	}
	if cfg.Watcher.DedupWindow == 0 {
		cfg.Watcher.DedupWindow = Duration(10 * time.Minute)
	}
	if cfg.Watcher.Hostname == "" {
		if hn, err := os.Hostname(); err == nil {
			cfg.Watcher.Hostname = hn
		}
	}
	for i := range cfg.Checks {
		if cfg.Checks[i].Timeout == 0 {
			cfg.Checks[i].Timeout = Duration(3 * time.Second)
		}
		if cfg.Checks[i].Group == "" {
			cfg.Checks[i].Group = cfg.Relay.DefaultGroup
		}
		if cfg.Checks[i].Severity == "" {
			cfg.Checks[i].Severity = "warning"
		}
		if cfg.Checks[i].Interval == 0 {
			cfg.Checks[i].Interval = cfg.Watcher.Interval
		}
	}
}

func validate(cfg *Config) error {
	if cfg.Relay.Endpoint == "" {
		return errors.New("relay.endpoint is required")
	}
	if cfg.Relay.Key == "" {
		return errors.New("relay.key is required")
	}
	if cfg.Relay.DefaultGroup == "" {
		return errors.New("relay.default_group is required")
	}
	if len(cfg.Checks) == 0 {
		return errors.New("at least one check is required")
	}

	seen := make(map[string]struct{}, len(cfg.Checks))
	for i, ch := range cfg.Checks {
		if ch.Name == "" {
			return fmt.Errorf("checks[%d].name is required", i)
		}
		if _, ok := seen[ch.Name]; ok {
			return fmt.Errorf("duplicate check name: %s", ch.Name)
		}
		seen[ch.Name] = struct{}{}

		switch ch.Type {
		case "http":
			if ch.URL == "" {
				return fmt.Errorf("check %q: url is required", ch.Name)
			}
		case "tcp":
			if ch.Target == "" {
				return fmt.Errorf("check %q: target is required", ch.Name)
			}
			if _, _, err := net.SplitHostPort(ch.Target); err != nil {
				return fmt.Errorf("check %q: invalid target %q: %w", ch.Name, ch.Target, err)
			}
		case "systemd":
			if ch.Service == "" {
				return fmt.Errorf("check %q: service is required", ch.Name)
			}
		case "disk":
			if ch.Path == "" {
				return fmt.Errorf("check %q: path is required", ch.Name)
			}
			if ch.MaxUsedPercent <= 0 {
				return fmt.Errorf("check %q: max_used_percent must be > 0", ch.Name)
			}
		case "memory":
			if ch.MaxMemoryUsedPercent <= 0 {
				return fmt.Errorf("check %q: max_used_percent must be > 0", ch.Name)
			}
		case "tls_cert":
			if ch.Target == "" {
				return fmt.Errorf("check %q: target is required", ch.Name)
			}
			if ch.WarnDays <= 0 {
				return fmt.Errorf("check %q: warn_days must be > 0", ch.Name)
			}
			if ch.CriticalDays <= 0 {
				return fmt.Errorf("check %q: critical_days must be > 0", ch.Name)
			}
		default:
			return fmt.Errorf("check %q: unsupported type %q", ch.Name, ch.Type)
		}
	}
	return nil
}

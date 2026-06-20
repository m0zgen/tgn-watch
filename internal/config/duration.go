package config

import (
	"fmt"
	"strconv"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalYAML(value interface{}) error {
	switch v := value.(type) {
	case string:
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		*d = Duration(parsed)
		return nil
	case int:
		*d = Duration(time.Duration(v))
		return nil
	case int64:
		*d = Duration(time.Duration(v))
		return nil
	case float64:
		*d = Duration(time.Duration(v))
		return nil
	default:
		return fmt.Errorf("invalid duration value %q", value)
	}
}

func (d Duration) Duration() time.Duration { return time.Duration(d) }
func (d Duration) String() string          { return time.Duration(d).String() }

func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = 0
		return nil
	}
	if n, err := strconv.ParseInt(string(text), 10, 64); err == nil {
		*d = Duration(time.Duration(n))
		return nil
	}
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

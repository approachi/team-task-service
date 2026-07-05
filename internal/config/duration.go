package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration so config.yaml can use human-readable
// values like "10s" or "5m" instead of raw nanosecond integers.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

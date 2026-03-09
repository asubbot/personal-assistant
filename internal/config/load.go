package config

import "errors"

// Config holds application configuration (full struct defined in task 1.1).
type Config struct{}

// Load reads and validates config from path. Stub: not implemented until task 1.1.
func Load(path string) (*Config, error) {
	return nil, errors.New("config load not implemented")
}

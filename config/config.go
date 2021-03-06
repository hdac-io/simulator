package config

import (
	"time"
)

// Config contains various configuration
type Config struct {
	Consensus *consensusConfig
}

type consensusConfig struct {
	BlockTime time.Duration // Block time
	LenULB    int           // Length of unconfirmed leading blocks
}

// GetDefault retrieves default configuration
func GetDefault() *Config {
	c := consensusConfig{
		BlockTime: 1 * time.Second,
		LenULB:    2,
	}

	return &Config{
		Consensus: &c,
	}
}

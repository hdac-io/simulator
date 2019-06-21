package config

import (
	"time"

	"github.com/hdac-io/simulator/consensus"
)

// Config contains various configuration
type Config struct {
	Consensus     *consensusConfig
	NumValidators int // Number of validators
	NumNodes      int // Number of non-validator nodes
}

type consensusConfig struct {
	Consensus consensus.Consensus
	BlockTime time.Duration // Block time
	LenULB    int           // Length of unconfirmed leading blocks
}

// GetDefault retrieves default configuration
func GetDefault() *Config {
	c := consensusConfig{
		Consensus: consensus.Get("friday"),
		BlockTime: 1 * time.Second,
		LenULB:    2,
	}

	return &Config{
		Consensus:     &c,
		NumValidators: 3,
		NumNodes:      0,
	}
}

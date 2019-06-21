package config

import (
	"time"
)

type Config struct {
	Consensus *ConsensusConfig
}

func DefaultConfig() *Config {
	return &Config{
		Consensus: DefaultConsensusConfig(),
	}
}

func TestConfig() *Config {
	return &Config{
		Consensus: TestConsensusConfig(),
	}
}

type ConsensusConfig struct {
	BlockTime     time.Duration // Block time
	NumValidators int           // Number of validators
	LenULB        int           // Length of unconfirmed leading blocks
}

func DefaultConsensusConfig() *ConsensusConfig {
	return &ConsensusConfig{
		BlockTime: 1 * time.Second,
	}
}
func TestConsensusConfig() *ConsensusConfig {
	return &ConsensusConfig{
		BlockTime:     1 * time.Second,
		NumValidators: 3,
		LenULB:        2,
	}
}

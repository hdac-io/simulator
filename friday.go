package main

import (
	"sync"
	"time"

	simulcfg "github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/fridayconsensus"
	"github.com/hdac-io/simulator/network"
	log "github.com/inconshreveable/log15"
)

func main() {
	logger := log.New("module", "main")
	logger.Info("Initialize Validators and AddressBooks")

	cfg := simulcfg.TestConfig()
	validators := make([]*fridayconsensus.Validator, cfg.Consensus.NumValidators)
	addressbook := make([]*network.Network, cfg.Consensus.NumValidators)
	for id := range validators {
		validators[id] = fridayconsensus.NewValidator(id, cfg.Consensus.BlockTime, cfg.Consensus.NumValidators, cfg.Consensus.LenULB)
		addressbook[id] = validators[id].GetAddress()
	}

	var wg sync.WaitGroup
	wg.Add(len(validators))
	for _, validator := range validators {
		// Genesis time for testing
		genesisTime := time.Now().Add(1 * time.Second).Round(1 * time.Second)
		go validator.Start(genesisTime, addressbook, &wg)
	}

	wg.Wait()
}

package main

import (
	"sync"
	"time"

	config "github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/fridayconsensus"
	"github.com/hdac-io/simulator/network"
	log "github.com/inconshreveable/log15"
)

func main() {
	logger := log.New("module", "main")
	logger.Info("Initialize Validators and AddressBooks")

	config := config.TestConfig()
	validators := make([]*fridayconsensus.Validator, config.Consensus.NumValidators)
	addressbook := make([]*network.Network, config.Consensus.NumValidators)
	for id := range validators {
		validators[id] = fridayconsensus.NewValidator(id, config.Consensus.BlockTime, config.Consensus.NumValidators, config.Consensus.LenULB)
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

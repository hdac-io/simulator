package main

import (
	"github.com/hdac-io/simulator/fridayconsensus"
    "github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/util/log"
	"sync"
	"time"
)

// Block time
	logger := log.New("module", "main")
	logger.Info("Initialize Validators and AddressBooks")

const blockTime time.Duration = 1 * time.Second

// Number of validators
const numValidators int = 3

// Length of Uncompleted Leading Blocks
const lenULB = 2

func main() {
	validators := make([]*fridayconsensus.Validator, numValidators)
	addressbook := make([]*network.Network, numValidators)
	for id := range validators {
		validators[id] = fridayconsensus.NewValidator(id, blockTime, numValidators, lenULB)
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

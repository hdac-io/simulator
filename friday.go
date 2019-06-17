package main

import (
	"simulator/fridayconsensus"
	"simulator/network"
	"sync"
	"time"
)

// Block time
const blockTime time.Duration = 1 * time.Second

// Number of validators
const numValidators int = 3

// Length of unconfirmed leading blocks
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

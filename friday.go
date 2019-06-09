package main

import (
	"simulator/friday_consensus"
	"sync"
	"time"
)

func main() {
	const blockTime = 1 * time.Second
	const numValidator = 3
	validators := make([]*friday_consensus.Validator, numValidator)
	addressbook := make([]*friday_consensus.Channel, numValidator)
	for id, _ := range validators {
		validators[id] = friday_consensus.NewValidator(id, blockTime)
		addressbook[id] = validators[id].GetAddress()
	}

	var wg sync.WaitGroup
	wg.Add(len(validators))
	for _, validator := range validators {
		validator.SetAddressbook(addressbook)
		// Genesis time for testing
		genesisTime := time.Now().Add(1 * time.Second).Round(1 * time.Second)
		go validator.Start(genesisTime, &wg)
	}

	wg.Wait()
}
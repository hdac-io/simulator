package main

import (
	"simulator/fridayconsensus"
	"simulator/network"
	"sync"
	"time"
)

func main() {
	const blockTime time.Duration = 1 * time.Second
	const numValidator int = 3
	validators := make([]*fridayconsensus.Validator, numValidator)
	addressbook := make([]*network.Network, numValidator)
	for id := range validators {
		validators[id] = fridayconsensus.NewValidator(id, blockTime)
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

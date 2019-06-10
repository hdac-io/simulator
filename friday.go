package main

import (
	"simulator/fridayconsensus"
	"sync"
	"time"
)

func main() {
	const blockTime time.Duration = 1 * time.Second
	const numValidator int = 3
	validators := make([]*fridayconsensus.Validator, numValidator)
	addressbook := make([]*fridayconsensus.Channel, numValidator)
	for id := range validators {
		validators[id] = fridayconsensus.NewValidator(id, blockTime)
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

package main

import (
	"friday/simulator/consensus"
	"sync"
	"time"
)

func main() {
	const blockTime = 1 * time.Second
	const numValidator = 3
	validators := make([]*consensus.Validator, numValidator)
	addressbook := make([]*consensus.Channel, numValidator)
	for id, _ := range validators {
		validators[id] = consensus.NewValidator(id, blockTime)
		addressbook[id] = validators[id].GetAddress()
	}

	var wg sync.WaitGroup
	wg.Add(len(validators))
	for _, validator := range validators {
		validator.SetAddressbook(addressbook)
		go validator.Start(&wg)
	}

	wg.Wait()
}

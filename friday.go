package main

import (
	"sync"
	"time"

	"github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/node"
	log "github.com/inconshreveable/log15"
)

func main() {
	logger := log.New("module", "main")
	logger.Info("Initialize Validators and AddressBooks")

	config := config.GetDefault()
	nodes := make([]*node.Node, config.NumValidators+config.NumNodes)
	addressbook := make([]*network.Network, config.NumValidators+config.NumNodes)
	id := 0
	for id < config.NumValidators {
		nodes[id] = node.NewValidator(config.Consensus.Consensus, id, config.Consensus.BlockTime, config.NumValidators, config.Consensus.LenULB)
		addressbook[id] = nodes[id].GetAddress()
		id++
	}
	for id < config.NumValidators+config.NumNodes {
		nodes[id] = node.New(id)
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, node := range nodes {
		// Genesis time for testing
		genesisTime := time.Now().Add(1 * time.Second).Round(1 * time.Second)
		go node.Start(genesisTime, addressbook, &wg)
	}

	wg.Wait()
}

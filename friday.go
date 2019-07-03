package main

import (
	"sync"
	"time"

	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/node"
	log "github.com/inconshreveable/log15"
)

func main() {
	logger := log.New("module", "main")
	logger.Info("Initialize Validators and AddressBooks")

	//Initialize External BLS Package
	bls.Init(bls.CurveFp254BNb)

	config := config.GetDefault()
	nodes := make([]*node.Node, config.NumValidators+config.NumNodes)
	identities := make([]node.Identity, config.NumValidators+config.NumNodes)
	id := 0
	for id < config.NumValidators {
		nodes[id] = node.NewValidator(id, config.NumValidators, config.Consensus.LenULB, config.Consensus.BlockTime)
		identities[id] = nodes[id].GetIdentity()
		id++
	}
	for id < config.NumValidators+config.NumNodes {
		nodes[id] = node.New(id, config.NumValidators, config.Consensus.LenULB)
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, node := range nodes {
		// Genesis time for testing
		genesisTime := time.Now().Add(1 * time.Second).Round(1 * time.Second)
		go node.Start(genesisTime, identities, &wg)
	}

	wg.Wait()
}

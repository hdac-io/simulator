package main

import (
	"net"
	"os"
	"sync"
	"time"

	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/node"
	"github.com/hdac-io/simulator/node/status"
	log "github.com/inconshreveable/log15"
)

const defaultIP = "127.0.0.1"

func main() {
	// Set my TCP address
	address := defaultIP
	if len(os.Args) > 1 {
		address = os.Args[1]
	}
	nodeAddress, err := net.ResolveTCPAddr("tcp", address+":0")
	if err != nil {
		panic(err)
	}

	// Set default genesis time for local test
	genesisTime := time.Now().Add(5 * time.Second).Round(1 * time.Second)
	if len(os.Args) > 2 {
		time, err := time.Parse(time.RFC3339, os.Args[2])
		if err != nil {
			panic(err)
		}
		genesisTime = time
	}

	// Initialize external BLS package
	bls.Init(bls.CurveFp254BNb)

	logger := log.New("module", "main")
	logger.Info("Node information", "IP address", nodeAddress.IP, "Genesis time", genesisTime.String())
	logger.Info("Initialize validators and addressbook")

	// prepare addressbook
	addressbook := node.PrepareAddressbook()

	config := config.GetDefault()
	nodes := make([]*node.Node, 0)
	for _, address := range addressbook {
		ip, err := net.ResolveTCPAddr("tcp", address.Address.(string))
		if err != nil {
			panic(err)
		}
		if ip.IP.Equal(nodeAddress.IP) {
			// FIXME: we should copy addressbook for runtime modification by nodes
			nodes = append(nodes, node.NewValidator(address.ID, addressbook, config.Consensus.LenULB, config.Consensus.BlockTime))
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, node := range nodes {
		// Genesis time for testing
		go node.Start(genesisTime, &wg)
	}

	// For analysis, do not wait this goroutine
	startAnalyze(logger, genesisTime)

	wg.Wait()
}

func startAnalyze(logger log.Logger, genesisTime time.Time) {
	status.Analysis.Enabled = true
	status.Analysis.FastestFinalizedTime = time.Duration(10) * time.Second

	go func() {
		// Wait for genesis time
		time.Sleep(genesisTime.Sub(time.Now()))

		for {
			time.Sleep(5 * time.Second)
			status.Analysis.Lock()
			logger.Crit("Fastest finalized time", "time", status.Analysis.FastestFinalizedTime)
			logger.Crit("Laziest finalized time", "time", status.Analysis.LaziestFinalizedTime)
			logger.Crit("Average finalized time", "time", status.Analysis.AverageFinalizedTime)
			status.Analysis.Unlock()
		}
	}()
}

package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/node"
	log "github.com/inconshreveable/log15"
)

const defaultIP = "127.0.0.1:0"

func main() {
	// Set my TCP address
	address := defaultIP
	if len(os.Args) > 1 {
		address = os.Args[1]
	}
	myIP, err := net.ResolveTCPAddr("tcp", address)
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
		fmt.Println(time)
		genesisTime = time
	}

	// Initialize external BLS package
	bls.Init(bls.CurveFp254BNb)

	logger := log.New("module", "main")
	logger.Info("Initialize Validators and Addressbooks")

	// prepare addressbook
	addressbook := node.PrepareAddressbook()

	config := config.GetDefault()
	nodes := make([]*node.Node, 0)
	for _, address := range addressbook {
		ip, err := net.ResolveTCPAddr("tcp", address.Address.(string))
		if err != nil {
			panic(err)
		}
		if ip.IP.Equal(myIP.IP) {
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

	wg.Wait()
}

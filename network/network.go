package network

import (
	"math/rand"
	"time"
)

func randomDelay(min int, max int) func() time.Duration {
	seed := int64(time.Now().Nanosecond())
	random := rand.New(rand.NewSource(seed))
	return func() time.Duration {
		return time.Duration(min+random.Intn(max-min)) * time.Millisecond
	}
}

type load = interface{}

// Network represents virtual public network
type Network struct {
	network  chan load
	getDelay func() time.Duration
}

// NewNetwork construct Network
func NewNetwork() *Network {
	return &Network{
		network: make(chan load, 1024),
		// Simulated network delay: 50ms ~ 250ms, for instant finalization
		// getDelay: randomDelay(50, 250),
		// Simulated network delay: 50ms ~ 600ms, for one block delayed finalization
		// getDelay: randomDelay(50, 600),
		// Simulated network delay: 50ms ~ 950ms, for two block delayed finalization
		getDelay: randomDelay(50, 950),
	}
}

// Write load to virtual network
func (n *Network) Write(l load) {
	go func() {
		time.Sleep(n.getDelay())
		n.network <- l
	}()
}

// Read load from virtual network
func (n *Network) Read() load {
	return <-n.network
}

package network

import (
	"math/rand"
	"time"
)

// Simulated network delay: 50ms ~ 250ms, for instant finalization
// Simulated network delay: 50ms ~ 550ms, for one block delayed finalization
// Simulated network delay: 50ms ~ 850ms, for two block delayed finalization
const delayMin = 50
const delayMax = 850

func randomDelay() func() time.Duration {
	seed := int64(time.Now().Nanosecond())
	random := rand.New(rand.NewSource(seed))
	return func() time.Duration {
		return time.Duration(delayMin+random.Intn(delayMax-delayMin)) * time.Millisecond
	}
}

type load = interface{}

// Network represents virtual public network
type Network struct {
	Unique   int
	inbound  chan load
	outbound chan load
	getDelay func() time.Duration
}

// NewNetwork returns networ
func NewNetwork(address Address) Network {
	return Network{
		Unique:   address.Unique,
		inbound:  make(chan load, 1024),
		outbound: make(chan load, 1024),
		getDelay: randomDelay(),
	}
}

// NewLoopbackNetwork returns networ
func NewLoopbackNetwork(address Address) Network {
	c := make(chan load, 1024)
	return Network{
		Unique:   address.Unique,
		inbound:  c,
		outbound: c,
		getDelay: randomDelay(),
	}
}

// Write load to virtual network
func (n *Network) Write(l load) {
	go func() {
		time.Sleep(n.getDelay())
		n.outbound <- l
	}()
}

// Read load from virtual network
func (n *Network) Read() load {
	return <-n.inbound
}

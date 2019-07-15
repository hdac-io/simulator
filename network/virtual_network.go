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

// VirtualNetwork represents virtual public network
type virtualNetwork struct {
	address  Address
	inbound  chan load
	outbound chan load
	getDelay func() time.Duration
}

func newVirtualNetwork(address Address) *virtualNetwork {
	return &virtualNetwork{
		address:  address,
		inbound:  make(chan load, 1024),
		outbound: make(chan load, 1024),
		getDelay: randomDelay(),
	}
}

func newVirtualLoopbackNetwork(address Address) *virtualNetwork {
	c := make(chan load, 1024)
	return &virtualNetwork{
		address:  address,
		inbound:  c,
		outbound: c,
		getDelay: randomDelay(),
	}
}

// Read load from virtual network
func (n *virtualNetwork) Read() load {
	return <-n.inbound
}

// Write load to virtual network
func (n *virtualNetwork) Write(l load) {
	go func() {
		time.Sleep(n.getDelay())
		n.outbound <- l
	}()
}

// GetAddress retrieves network address
func (n *virtualNetwork) GetAddress() Address {
	return n.address
}

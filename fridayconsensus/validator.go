package fridayconsensus

import (
	"math/rand"
	"simulator/network"
	"simulator/util"
	"sync"
	"time"
)

// Validator represents validator node
type Validator struct {
	id          int
	blockTime   time.Duration
	getRandom   func() int
	peer        *channel
	addressbook []*network.Network
	blocks      []block
}

func randomSignature(unique int, max int) func() int {
	seed := int64(time.Now().Nanosecond() + unique)
	random := rand.New(rand.NewSource(seed))
	return func() int {
		return random.Intn(max)
	}
}

// NewValidator construct Validator
func NewValidator(id int, blockTime time.Duration) *Validator {
	return &Validator{
		id:        id,
		blockTime: blockTime,
		peer:      newChannel(),
		blocks:    make([]block, 0, 1024),
	}
}

// GetAddress returns validator's inbound address
func (v *Validator) GetAddress() *network.Network {
	return v.peer.inbound.network
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	v.getRandom = randomSignature(v.id, len(addressbook))
	v.addressbook = addressbook
	v.peer.start(addressbook)

	// Add pre-genesis block
	v.blocks = append(v.blocks, block{
		height: 0,
		// pre-genesis block timestamp is set "blockTime" before genesis time
		timestamp: genesisTime.Add(-(1 * time.Second)).UnixNano(),
	})

	for {
		recentBlock := &v.blocks[len(v.blocks)-1]

		// For genesis block and next one
		next := 0
		if recentBlock.height >= 2 {
			// calculation
			sum := 0
			for _, value := range recentBlock.signatures {
				sum += value
			}
			next = sum % (len(v.addressbook))
		}

		signatures := make([]int, len(v.addressbook))
		if recentBlock.height >= 1 {
			for index := range signatures {
				signatures[index] = v.peer.readSignature()
			}
		}

		if next == v.id {
			// My turn
			now := time.Now()
			nextBlockTime := time.Unix(0, recentBlock.timestamp).Add(v.blockTime)
			time.Sleep(nextBlockTime.Sub(now))

			// Produce new block
			newBlock := block{
				height:     recentBlock.height + 1,
				timestamp:  nextBlockTime.UnixNano(),
				producer:   v.id,
				signatures: signatures,
			}

			// Send new block
			v.peer.sendBlock(newBlock)
			util.Log("#", v.id, "Block produced\n", newBlock)
		} else {
			// Not my turn
		}
		block := v.peer.readBlock()
		v.receiveBlock(block)
	}
}

func (v *Validator) stop() {
	// Clean validator up
}

func (v *Validator) receiveBlock(b block) {
	// Validation
	if !v.validate() {
		return
	}

	// pre-commit
	v.preCommit()

	// commit
	v.commit(b)

	util.Log("#", v.id, "Block received and committed. Blockheight =", b.height)
}

func (v *Validator) preCommit() {
	// generate random signature
	sig := v.getRandom()

	// send piece to others
	v.peer.sendSignature(sig)
}

func (v *Validator) commit(b block) {
	v.blocks = append(v.blocks, b)
}

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}

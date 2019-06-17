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
	parameter       parameter
	id              int
	getRandom       func() int
	peer            *channel
	addressbook     []*network.Network
	blocks          []block
	blocksCandidate []block
	pool            signaturepool
	// Is this necessary ??
	signatures [][]signature
}

type parameter struct {
	numValidators int
	lenULB        int
	blockTime     time.Duration
}

func randomSignature(unique int, max int) func() int {
	seed := int64(time.Now().Nanosecond() + unique)
	random := rand.New(rand.NewSource(seed))
	return func() int {
		return random.Intn(max)
	}
}

// NewValidator construct Validator
func NewValidator(id int, blockTime time.Duration, numValidators int, lenULB int) *Validator {
	parameter := parameter{
		numValidators: numValidators,
		lenULB:        lenULB,
		blockTime:     blockTime,
	}
	return &Validator{
		parameter: parameter,
		id:        id,
		peer:      newChannel(),
		blocks:    make([]block, 0, 1024),
		pool:      newSignaturePool(),
	}
}

// GetAddress returns validator's inbound address
func (v *Validator) GetAddress() *network.Network {
	return v.peer.inbound.network
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	// Prepare validator
	v.getRandom = randomSignature(v.id, len(addressbook))
	v.addressbook = addressbook
	// Start channel
	v.peer.start(addressbook)

	// Start producing loop
	go v.produceLoop(genesisTime)

	// Start validating loop
	go v.validationLoop()

	// Start receiving loop
	v.receiveLoop()
}

func (v *Validator) produceLoop(genesisTime time.Time) {
	nextBlockTime := genesisTime
	for {
		time.Sleep(nextBlockTime.Sub(time.Now()))
		nextBlockTime = v.produce(nextBlockTime)
	}
}

func (v *Validator) validationLoop() {
	if v.parameter.lenULB == 0 {
		for {
			block := v.peer.readBlock()
			v.validateBlock(block)
		}
	} else {
		for {
			block := v.peer.readBlock()
			go v.validateBlock(block)
		}
	}
}

func (v *Validator) receiveLoop() {
	for {
		signature := v.peer.readSignature()
		v.pool.add(signature)
	}
}

func (v *Validator) produce(nextBlockTime time.Time) time.Time {
	// Calculation
	next := 0
	if len(v.blocks) >= v.parameter.lenULB+1 {
		if v.parameter.lenULB == 0 {
			// We should use recent signatures
			next = v.getRandomNumberFromSignatures(v.signatures[len(v.signatures)-1])
		} else {
			// We can use calculated number written in block
			next = v.getFinalizedBlock().chosenNumber
		}
	} else {
		// Node 0 will producing
	}

	if next != v.id {
		// Not my turn
	} else {
		// My turn
		signatures := []signature{}
		if len(v.signatures) >= 1 {
			signatures = v.signatures[len(v.signatures)-1]
		}

		// Produce new block
		newBlock := block{
			height:       v.getRecentBlock().height + 1,
			timestamp:    nextBlockTime.UnixNano(),
			producer:     v.id,
			chosenNumber: v.getRandomNumberFromSignatures(signatures),
		}

		// Send new block
		v.peer.sendBlock(newBlock)
		util.Log("#", v.id, "Block produced\n", newBlock)
	}

	return nextBlockTime.Add(v.parameter.blockTime)
}

func (v *Validator) validateBlock(b block) {
	// Validation
	if !v.validate() {
		return
	}
	util.Log("#", v.id, "Block received. Blockheight =", b.height)

	// prepare
	v.prepare(b)

	// commit
	v.commit(b)

	util.Log("#", v.id, "Block committed. Blockheight =", b.height)
}

func (v *Validator) prepare(b block) {
	v.blocksCandidate = append(v.blocksCandidate, b)

	// Generate random signature
	sig := newSignature(v.id, b.height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sig)

	// Collect signatues
	// FIXME: naive implementation
	for {
		signatures := v.pool[b.height]
		if len(signatures) < v.parameter.numValidators {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		break
	}
	v.signatures = append(v.signatures, v.pool.remove(b.height))
}

func (v *Validator) commit(b block) {
	v.blocks = append(v.blocks, b)
}

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}

func (v *Validator) getRecentBlock() *block {
	recentBlock := &block{}
	if len(v.blocksCandidate) > len(v.blocks) {
		recentBlock = &v.blocksCandidate[len(v.blocksCandidate)-1]
	} else if len(v.blocks) >= 1 {
		recentBlock = &v.blocks[len(v.blocks)-1]
	}

	return recentBlock
}

func (v *Validator) getFinalizedBlock() block {
	return v.blocks[len(v.blocks)-(v.parameter.lenULB)]
}

func (v *Validator) getRandomNumberFromSignatures(sig []signature) int {
	sum := 0
	for _, value := range sig {
		sum += value.number
	}
	return sum % (v.parameter.numValidators)
}

func (v *Validator) stop() {
	// Clean validator up
}

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
	pool            *signaturepool
	signatures      [][]signature
	finalizedHeight int
	completedHeight int
	height          int
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
	v := &Validator{
		parameter: parameter,
		id:        id,
		peer:      newChannel(),
		blocks:    make([]block, 0, 1024),
		pool:      newSignaturePool(),
	}

	// Add dummy block
	v.blocks = append(v.blocks, block{})
	// Add dummy signatures
	v.signatures = append(v.signatures, []signature{})

	return v
}

// GetAddress returns validator's inbound address
func (v *Validator) GetAddress() *network.Network {
	return v.peer.inbound.network
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	// Prepare validator
	v.getRandom = randomSignature(v.id, v.parameter.numValidators)
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
	signatures := v.signatures[v.completedHeight]
	chosenNumber := v.getRandomNumberFromSignatures(signatures)

	// next := 0 if there is no completed block
	next := chosenNumber

	if next != v.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := block{
			height:       v.height + 1,
			timestamp:    nextBlockTime.UnixNano(),
			producer:     v.id,
			chosenNumber: chosenNumber,
		}

		// Pre-prepare / send new block
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
	v.height = b.height
	if v.height > v.parameter.lenULB {
		v.completedHeight = v.height - v.parameter.lenULB
	} else {
		// 0 means there is no completed block
		v.completedHeight = 0
	}
	v.blocks = append(v.blocks, b)
	util.Log("#", v.id, "Block received. Blockheight =", b.height)

	// prepare
	v.prepare(b)
	util.Log("#", v.id, "Block prepared. Blockheight =", b.height)

	// commit
	v.commit(b)
	util.Log("#", v.id, "Block finalized. Blockheight =", b.height)
}

func (v *Validator) prepare(b block) {
	// Generate random signature
	sig := newSignature(v.id, b.height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sig)

	// Collect signatues
	v.pool.waitAndRemove(b.height, v.parameter.numValidators)
}

func (v *Validator) commit(b block) {
	// Generate random signature
	sig := newSignature(v.id, b.height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sig)

	// Collect signatues
	sigs := v.pool.waitAndRemove(b.height, v.parameter.numValidators)
	v.signatures = append(v.signatures, sigs)
	v.finalizedHeight = b.height
}

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}

func (v *Validator) getRecentBlock() block {
	return v.blocks[v.height]
}

func (v *Validator) getFinalizedBlock() block {
	return v.blocks[v.finalizedHeight]
}

func (v *Validator) getCompletedBlock() block {
	return v.blocks[v.completedHeight]
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

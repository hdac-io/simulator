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
	numValidators int
	lenULB        int
	id            int
	blockTime     time.Duration
	getRandom     func() int
	peer          *channel
	addressbook   []*network.Network
	blocks        []block
	signatures    []signature
}

type signature struct {
	signatures []int
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
	return &Validator{
		numValidators: numValidators,
		lenULB:        lenULB,
		id:            id,
		blockTime:     blockTime,
		peer:          newChannel(),
		blocks:        make([]block, 0, 1024),
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

func (v *Validator) receiveLoop() {
	for {
		block := v.peer.readBlock()
		go v.validateBlock(block)
	}
}

func (v *Validator) produce(nextBlockTime time.Time) time.Time {
	// Calculation
	next := 0
	if len(v.blocks) >= v.lenULB {
		if v.lenULB == 1 {
			// We should use recent signatures
			next = v.getNumberFromSignatures(v.signatures[len(v.signatures)-1])
		} else {
			// We can use calculated number written in block
			next = v.getConfirmedBlock().chosenNumber
		}
	} else {
		// Node 0 will producing
	}

	if next != v.id {
		// Not my turn
	} else {

		// My turn
		signatures := signature{}
		if len(v.signatures) >= 1 {
			signatures = v.signatures[len(v.signatures)-1]
		}

		// Produce new block
		newBlock := block{
			height:       v.getRecentBlock().height + 1,
			timestamp:    nextBlockTime.UnixNano(),
			producer:     v.id,
			chosenNumber: v.getNumberFromSignatures(signatures),
		}

		// Send new block
		v.peer.sendBlock(newBlock)
		util.Log("#", v.id, "Block produced\n", newBlock)
	}

	return nextBlockTime.Add(v.blockTime)
}

func (v *Validator) stop() {
	// Clean validator up
}

func (v *Validator) validateBlock(b block) {
	// Validation
	if !v.validate() {
		return
	}

	// pre-commit
	v.preCommit(b)

	// commit
	v.commit(b)

	util.Log("#", v.id, "Block received and committed. Blockheight =", b.height)
}

func (v *Validator) preCommit(b block) {
	// Generate random signature
	sig := v.getRandom()

	// Send piece to others
	v.peer.sendSignature(sig)

	// Receive signatues
	signatures := signature{
		signatures: make([]int, v.numValidators),
	}
	for index := range signatures.signatures {
		signatures.signatures[index] = v.peer.readSignature()
	}
	v.signatures = append(v.signatures, signatures)
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
	if len(v.blocks) >= 1 {
		recentBlock = &v.blocks[len(v.blocks)-1]
	}

	return recentBlock
}

func (v *Validator) getConfirmedBlock() block {
	return v.blocks[len(v.blocks)-(v.lenULB-1)]
}

func (v *Validator) getNumberFromSignatures(sig signature) int {
	sum := 0
	for _, value := range sig.signatures {
		sum += value
	}
	return sum % (v.numValidators)
}

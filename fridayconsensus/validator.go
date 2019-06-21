package fridayconsensus

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/persistent"
	"github.com/hdac-io/simulator/signature"
	log "github.com/inconshreveable/log15"
)

// Validator represents validator node
type Validator struct {
	// For synchronization
	sync.Mutex
	cond         *sync.Cond
	waitFinalize bool

	// Chain parameters
	parameter parameter

	// Validator data
	id              int
	finalizedHeight int
	confirmedHeight int
	height          int
	next            int
	getRandom       func() int
	blocks          []block.Block

	// Network data
	peer        *channel
	addressbook []*network.Network

	// Transaction pool
	pool *signaturepool

	// Persistent
	persistent persistent.Persistent

	// Logger
	logger log.Logger
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

// NewValidator constructs Validator
func NewValidator(id int, blockTime time.Duration, numValidators int, lenULB int) *Validator {
	parameter := parameter{
		numValidators: numValidators,
		lenULB:        lenULB,
		blockTime:     blockTime,
	}
	v := &Validator{
		waitFinalize: false,
		parameter:    parameter,
		id:           id,
		peer:         newChannel(),
		blocks:       make([]block.Block, 0),
		persistent:   persistent.New(),
		pool:         newSignaturePool(),
		logger:       log.New("Validator", id),
	}
	v.cond = sync.NewCond(v)

	return v
}

// GetAddress returns validator's inbound address
func (v *Validator) GetAddress() *network.Network {
	return v.peer.inbound.network
}

func (v *Validator) initialize(addressbook []*network.Network) bool {
	// Prepare validator
	v.getRandom = randomSignature(v.id, v.parameter.numValidators)
	v.addressbook = addressbook
	// Start channel
	v.peer.start(addressbook)

	//FIXME: more detail successful
	return true
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	if !v.initialize(addressbook) {
		panic("Initialization failed !")
	}

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
		v.pool.add(signature.Kind, signature)
	}
}

func (v *Validator) produce(nextBlockTime time.Time) time.Time {
	// Height adjustment
	v.height++
	// Negative number and 0 mean there is no confirmed block
	v.confirmedHeight = v.height - (v.parameter.lenULB + 1)

	// Calculation
	signatures := v.getRecentConfirmedSignature()
	chosenNumber := v.getRandomNumberFromSignatures(signatures)

	// next := 0 if there is no completed block
	v.next = chosenNumber

	if v.next != v.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := block.New(v.height, nextBlockTime.UnixNano(), v.id)

		// Pre-prepare / send new block
		v.peer.sendBlock(newBlock)
		v.logger.Info("Block produced", "Height", newBlock.Height, "Producer", newBlock.Producer,
			"Timestmp", time.Unix(0, newBlock.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash))

	}

	return nextBlockTime.Add(v.parameter.blockTime)
}

func (v *Validator) validateBlock(b block.Block) {
	// Validation
	if !v.validate(b) {
		panic("There shoud be no byzitine nodes !")
		//return
	}

	v.logger.Info("Block received", "Blockheight", b.Height)
	v.Lock()
	if v.height != b.Height {
		panic("Block height is mismatch !")
	}
	v.blocks = append(v.blocks, b)
	v.Unlock()

	// Prepare
	v.prepare(b)
	v.logger.Info("Block prepared", "Blockheight", b.Height)

	// Commit / finalize
	v.finalize(b)
	v.logger.Info("Block finalized", "Blockheight", b.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (v *Validator) validate(b block.Block) bool {
	// Validate producer
	if v.next != b.Producer {
		return false
	}

	// Validate block hash
	if !bytes.Equal(b.Hash, block.CalculateHashFromBlock(b)) {
		return false
	}

	return true
}

func (v *Validator) prepare(b block.Block) {
	// Generate dummy signature with -1
	sign := signature.New(v.id, signature.Prepare, b.Height, -1)

	// Send piece to others
	v.peer.sendSignature(sign)

	// Collect signatues
	v.pool.waitAndRemove(signature.Prepare, b.Height, v.parameter.numValidators)
}

func (v *Validator) finalize(b block.Block) {
	// Generate random signature
	sign := signature.New(v.id, signature.Commit, b.Height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sign)

	// Collect signatues
	signs := v.pool.waitAndRemove(signature.Commit, b.Height, v.parameter.numValidators)

	// Finalize
	v.Lock()

	for b.Height > v.finalizedHeight+1 {
		v.logger.Warn("Previous block is not finalized yet !", "Current Finalizing height", b.Height, "Previous finalized height", v.finalizedHeight)
		v.waitFinalize = true
		v.cond.Wait()
	}
	v.finalizedHeight = b.Height
	if v.waitFinalize {
		v.cond.Broadcast()
		v.waitFinalize = false
	}

	// Store finalized block
	v.persistent.AddBlock(b)
	v.blocks = v.blocks[1:]
	// Store finalized signature
	v.persistent.AddSignature(signs)

	v.Unlock()
}

func (v *Validator) getCurrentBlock() block.Block {
	b := v.getRecentBlock()
	if b.Height != v.height {
		return block.Block{Height: -1}
	}

	return b
}

func (v *Validator) getRecentBlock() block.Block {
	return v.blocks[len(v.blocks)-1]
}

func (v *Validator) getRecentFinalizedBlock() block.Block {
	return v.persistent.GetBlock(v.finalizedHeight)
}

func (v *Validator) getRecentConfirmedBlock() block.Block {
	return v.persistent.GetBlock(v.confirmedHeight)
}

func (v *Validator) getRecentConfirmedSignature() []signature.Signature {
	return v.persistent.GetSignature(v.confirmedHeight)
}

func (v *Validator) getRandomNumberFromSignatures(signs []signature.Signature) int {
	sum := 0
	for _, sign := range signs {
		if sign.Number < 0 {
			panic("Bad signature !")
		}
		sum += sign.Number
	}
	return sum % (v.parameter.numValidators)
}

func (v *Validator) stop() {
	// Clean validator up
}

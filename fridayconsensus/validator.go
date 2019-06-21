package fridayconsensus

import (
	"math/rand"
	"sync"
	"time"

	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/persistent"
	"github.com/hdac-io/simulator/signature"
	"github.com/hdac-io/simulator/types"
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
	getRandom       func() int
	blocks          []types.Block

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
		blocks:       make([]types.Block, 0),
		persistent:   persistent.NewPersistent(),
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
	next := chosenNumber

	if next != v.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := types.Block{
			Height:       v.height,
			Timestamp:    nextBlockTime.UnixNano(),
			Producer:     v.id,
			ChosenNumber: chosenNumber,
		}

		// Pre-prepare / send new block
		v.peer.sendBlock(newBlock)
		v.logger.Info("Block produced", "Height", newBlock.Height, "Producer", newBlock.Producer,
			"ChosenNumber", newBlock.ChosenNumber, "Timestmp", time.Unix(0, newBlock.Timestamp))

	}

	return nextBlockTime.Add(v.parameter.blockTime)
}

func (v *Validator) validateBlock(b types.Block) {
	// Validation
	if !v.validate() {
		panic("There shoud be no byzitine nodes !")
		//return
	}

	v.Lock()
	if v.height != b.Height {
		panic("Block height is mismatch !")
	}
	v.blocks = append(v.blocks, b)
	v.Unlock()
	v.logger.Info("Block received", "Blockheight", b.Height)

	// Prepare
	v.prepare(b)
	v.logger.Info("Block prepared", "Blockheight", b.Height)

	// Commit / finalize
	v.finalize(b)
	v.logger.Info("Block finalized", "Blockheight", b.Height)
}

func (v *Validator) prepare(b types.Block) {
	// Generate dummy signature with -1
	sign := signature.NewSignature(v.id, signature.Prepare, b.Height, -1)

	// Send piece to others
	v.peer.sendSignature(sign)

	// Collect signatues
	v.pool.waitAndRemove(signature.Prepare, b.Height, v.parameter.numValidators)
}

func (v *Validator) finalize(b types.Block) {
	// Generate random signature
	sign := signature.NewSignature(v.id, signature.Commit, b.Height, v.getRandom())

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

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}

func (v *Validator) getCurrentBlock() types.Block {
	b := v.getRecentBlock()
	if b.Height != v.height {
		return types.Block{Height: -1}
	}

	return b
}

func (v *Validator) getRecentBlock() types.Block {
	return v.blocks[len(v.blocks)-1]
}

func (v *Validator) getRecentFinalizedBlock() types.Block {
	return v.persistent.GetBlock(v.finalizedHeight)
}

func (v *Validator) getRecentConfirmedBlock() types.Block {
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

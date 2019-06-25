package node

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"

	"github.com/hdac-io/simulator/node/status"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/persistent"
	"github.com/hdac-io/simulator/signature"
	log "github.com/inconshreveable/log15"
)

// Node represents validator node
type Node struct {
	// Chain parameters
	parameter parameter

	// Validator data
	id        int
	validator bool
	next      int

	// Peer-to-peer network
	peer *channel

	// Status
	status *status.Status

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

// New constructs node
func New(id int, numValidators, lenULB int) *Node {
	parameter := parameter{
		numValidators: numValidators,
		lenULB:        lenULB,
	}

	v := &Node{
		id:         id,
		peer:       newChannel(),
		parameter:  parameter,
		persistent: persistent.New(),
		pool:       newSignaturePool(),
		logger:     log.New("Validator", id),
	}
	v.status = status.New(id, numValidators, v.logger)

	return v
}

// NewValidator constructs validator node
func NewValidator(id int, numValidators int, lenULB int, blockTime time.Duration) *Node {
	v := New(id, numValidators, lenULB)
	v.validator = true
	v.parameter.blockTime = blockTime

	return v
}

// GetAddress returns validator's inbound address
func (v *Node) GetAddress() *network.Network {
	return v.peer.inbound.network
}

func (v *Node) initialize(addressbook []*network.Network) bool {
	// Start channel
	v.peer.start(addressbook)

	//FIXME: more detail successful
	return true
}

// Start starts validator with genesis time
func (v *Node) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	if !v.initialize(addressbook) {
		panic("Initialization failed !")
	}

	// Wait for genesis
	time.Sleep(genesisTime.Sub(time.Now()))

	if v.validator {
		// Start producing loop
		go v.produceLoop(genesisTime)

		// Start validating loop
		go v.validationLoop()
	}

	// Start receiving loop
	v.receiveLoop()
}

func (v *Node) produceLoop(genesisTime time.Time) {
	nextBlockTime := genesisTime
	for {
		time.Sleep(nextBlockTime.Sub(time.Now()))
		nextBlockTime = v.produce(nextBlockTime)
	}
}

func (v *Node) validationLoop() {
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

func (v *Node) receiveLoop() {
	for {
		signature := v.peer.readSignature()
		v.pool.add(signature.Kind, signature)
	}
}

func (v *Node) produce(nextBlockTime time.Time) time.Time {
	// Calculation
	signatures := v.status.GetRecentConfirmedSignature()
	chosenNumber := v.getRandomNumberFromSignatures(signatures)

	// next := 0 if there is no completed block
	v.next = chosenNumber

	if v.next != v.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := block.New(v.status.GetHeight()+1, nextBlockTime.UnixNano(), v.id)

		// Pre-prepare / send new block
		v.peer.sendBlock(newBlock)
		v.logger.Info("Block produced", "Height", newBlock.Height, "Producer", newBlock.Producer,
			"Timestmp", time.Unix(0, newBlock.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash))

	}

	return nextBlockTime.Add(v.parameter.blockTime)
}

func (v *Node) validateBlock(b block.Block) {
	// Validation
	if !v.validate(b) {
		panic("There shoud be no byzitine nodes !")
		//return
	}

	v.logger.Info("Block received", "Blockheight", b.Height)
	v.status.AppendBlock(b)

	// Prepare
	v.prepare(b)
	v.logger.Info("Block prepared", "Blockheight", b.Height)

	// Commit / finalize
	v.finalize(b)
	v.logger.Info("Block finalized", "Blockheight", b.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (v *Node) validate(b block.Block) bool {
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

func (v *Node) prepare(b block.Block) {
	// Generate dummy signature with -1
	sign := signature.New(v.id, signature.Prepare, b.Height, -1)

	// Send piece to others
	v.peer.sendSignature(sign)

	// Collect signatues
	v.pool.waitAndRemove(signature.Prepare, b.Height, v.parameter.numValidators)
}

func (v *Node) finalize(b block.Block) {
	// Generate random signature
	sign := signature.New(v.id, signature.Commit, b.Height, v.status.GetRandom())

	// Send piece to others
	v.peer.sendSignature(sign)

	// Collect signatues
	signs := v.pool.waitAndRemove(signature.Commit, b.Height, v.parameter.numValidators)

	// Finalize
	v.status.Finalize(b, signs)
}

func (v *Node) stop() {
	// Clean validator up
}

func (v *Node) getRandomNumberFromSignatures(signs []signature.Signature) int {
	sum := 0
	for _, sign := range signs {
		if sign.Number < 0 {
			panic("Bad signature !")
		}
		sum += sign.Number
	}
	return sum % (v.parameter.numValidators)
}

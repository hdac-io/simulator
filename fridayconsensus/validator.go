package fridayconsensus

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/google/keytransparency/core/crypto/vrf"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/types"
	log "github.com/inconshreveable/log15"
)

// Validator represents validator node
type Validator struct {
	parameter       parameter
	id              int
	getRandom       func() int
	peer            *channel
	addressbook     []*network.Network
	blocks          []types.Block
	pool            *signaturepool
	signatures      [][]types.Signature
	finalizedHeight int
	completedHeight int
	height          int
	logger          log.Logger
	privKey         vrf.PrivateKey
	pubKey          vrf.PublicKey
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
		blocks:    make([]types.Block, 0, 1024),
		pool:      newSignaturePool(),
		logger:    log.New("Validator", id),
	}

	// Initailze VRF Key Pair
	v.privKey, v.pubKey = p256.GenerateKey()

	// Add dummy block
	v.blocks = append(v.blocks, types.NewBlock(0,
		time.Now().Add(1*time.Second).Round(1*time.Second).UnixNano(),
		0,
		0))
	// Add dummy signatures
	v.signatures = append(v.signatures, []types.Signature{})

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

	// Produce Dummy VRFMeessage
	if v.id == 0 {
		v.produceBPSelectionByVRF(v.blocks[0])
	}

	//FIXME: more detail successful
	return true
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	if !v.initialize(addressbook) {
		v.logger.Error("failed initialization.")
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
		v.pool.add(signature)
	}
}

func (v *Validator) produceBPSelectionByVRF(producedBlock types.Block) {
	rand, proof := v.privKey.Evaluate(producedBlock.Hash[:])
	message := vrfMessage{
		rand:                   rand,
		proof:                  proof,
		previousProposerID:     v.id,
		previousProposerPubkey: v.pubKey,
		previousBlockHeight:    producedBlock.Header.Height,
	}
	v.peer.sendVRFMessage(message)
	v.logger.Info("VRF Produced")
}
func (v *Validator) calculateBPIDByVRF(message vrfMessage) (int, error) {
	if message.previousBlockHeight != v.height {
		return -1, errors.New("received previousBlockHeight is not equal then validator local height")
	}

	proofRand, err := message.previousProposerPubkey.ProofToHash(
		v.blocks[message.previousBlockHeight].Hash[:],
		message.proof)
	if proofRand != message.rand || err != nil {
		return -1, errors.New("verify failed of received rand into vrfMessage")
	}

	//TODO::check overflow
	so := int(binary.LittleEndian.Uint32(message.rand[:]))
	chosenNumber := so % v.parameter.numValidators
	v.logger.Debug("received vrf-rand to chosenNumber", "so", so, "chosenNumber", chosenNumber)

	return chosenNumber, nil
}

func (v *Validator) produce(nextBlockTime time.Time) time.Time {
	// Calculation block proposer
	vrfMessage := v.peer.readVRFMessage()

	chosenNumber, vrfErr := v.calculateBPIDByVRF(vrfMessage)
	if vrfErr != nil {
		v.logger.Crit(vrfErr.Error())
		//TODO::replace to decide next action when invalid VRF situation
		panic(vrfErr)
	}

	if chosenNumber != v.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := types.NewBlock(v.height+1, nextBlockTime.UnixNano(), v.id, chosenNumber)

		// Pre-prepare / send new block
		v.peer.sendBlock(newBlock)
		v.logger.Info("Block produced.",
			"Height", newBlock.Header.Height,
			"Producer", newBlock.Header.Producer,
			"ChosenNumber", newBlock.Header.ChosenNumber,
			"Hash", hex.EncodeToString(newBlock.Hash[:]),
			"Timestmp", time.Unix(0, newBlock.Header.Timestamp))

		// produce next block proposer selection by VRF
		go v.produceBPSelectionByVRF(newBlock)
	}

	return nextBlockTime.Add(v.parameter.blockTime)
}

func (v *Validator) validateBlock(b types.Block) {
	// Validation
	if !v.validate() {
		return
	}
	v.height = b.Header.Height
	if v.height > v.parameter.lenULB {
		v.completedHeight = v.height - v.parameter.lenULB
	} else {
		// 0 means there is no completed block
		v.completedHeight = 0
	}
	v.blocks = append(v.blocks, b)
	v.logger.Info("Block received.",
		"Blockheight", b.Header.Height)

	// prepare
	v.prepare(b)
	v.logger.Info("Block prepared.",
		"Blockheight", b.Header.Height)

	// commit
	v.finalize(b)
	v.logger.Info("Block finalized.",
		"Blockheight", b.Header.Height)
}

func (v *Validator) prepare(b types.Block) {
	// Generate random signature
	sig := newSignature(v.id, b.Header.Height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sig)

	// Collect signatues
	// TODO::FIXME timeout handling
	v.pool.waitAndRemove(b.Header.Height, v.parameter.numValidators)
}

func (v *Validator) finalize(b types.Block) {
	// Generate random signature
	sig := newSignature(v.id, b.Header.Height, v.getRandom())

	// Send piece to others
	v.peer.sendSignature(sig)

	// Collect signatues
	sigs := v.pool.waitAndRemove(b.Header.Height, v.parameter.numValidators)
	v.signatures = append(v.signatures, sigs)
	// Finalize
	v.finalizedHeight = b.Header.Height
}

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}

func (v *Validator) getRecentBlock() types.Block {
	return v.blocks[v.height]
}

func (v *Validator) getFinalizedBlock() types.Block {
	return v.blocks[v.finalizedHeight]
}

func (v *Validator) getCompletedBlock() types.Block {
	return v.blocks[v.completedHeight]
}

func (v *Validator) getRandomNumberFromSignatures(sig []types.Signature) int {
	sum := 0
	for _, value := range sig {
		sum += value.Number
	}

	return sum % (v.parameter.numValidators)
}

func (v *Validator) stop() {
	// Clean validator up
}

package node

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/signature"
	"github.com/hdac-io/simulator/vrfmessage"
)

type fridayVRF struct {
	node *Node
}

func newFridayVRF(node *Node) consensus {
	return &fridayVRF{node: node}
}

func (f *fridayVRF) start(genesisTime time.Time) {
	// Start producing loop
	go f.produceLoop(genesisTime)

	// Start validating loop
	go f.validationLoop()
}

func (f *fridayVRF) produceLoop(genesisTime time.Time) {
	nextBlockTime := genesisTime
	for {
		time.Sleep(nextBlockTime.Sub(time.Now()))
		nextBlockTime = f.produce(nextBlockTime)
	}
}

func (f *fridayVRF) validationLoop() {
	if f.node.parameter.lenULB == 0 {
		for {
			block := f.node.peer.readBlock()
			f.validateBlock(block)
		}
	} else {
		for {
			block := f.node.peer.readBlock()
			go f.validateBlock(block)
		}
	}
}

func (f *fridayVRF) makeVRFMessage(blockHash [32]byte, height int) vrfmessage.VRFMessage {
	rand, proof := f.node.privKey.Evaluate(blockHash[:])
	message := vrfmessage.VRFMessage{
		Rand:                   rand,
		Proof:                  proof,
		PreviousProposerID:     f.node.id,
		PreviousProposerPubkey: f.node.pubKey,
		PreviousBlockHeight:    height,
	}
	return message
}

func (f *fridayVRF) validateVRFMessage(message vrfmessage.VRFMessage) error {

	if message.PreviousBlockHeight != f.node.status.GetHeight()-1 {
		return errors.New("received previousBlockHeight is not equal then validator local height-1")
	}

	var targetHash [32]byte
	targetBlock, _ := f.node.status.GetBlock(message.PreviousBlockHeight)
	targetHash = targetBlock.Hash
	proofRand, err := message.PreviousProposerPubkey.ProofToHash(
		targetHash[:],
		message.Proof)
	if proofRand != message.Rand || err != nil {
		return errors.New("verify failed of received rand into vrfMessage")
	}

	return nil
}

func (f *fridayVRF) calculateBPIDByVRF(message vrfmessage.VRFMessage) int {
	//TODO::check overflow when based 32bit system
	so := int(binary.LittleEndian.Uint32(message.Rand[:]))
	chosenNumber := so % f.node.parameter.numValidators
	f.node.logger.Debug("received vrf-rand to chosenNumber", "so", so, "chosenNumber", chosenNumber)

	return chosenNumber
}

func (f *fridayVRF) getVRFMessage(blockHeight int) vrfmessage.VRFMessage {
	var vrfMessage vrfmessage.VRFMessage

	//getting VRFMessage by previous block body
	block, err := f.node.status.GetBlock(blockHeight)
	if err == nil {
		vrfMessage = block.VRF
	} else {
		panic("out-of-index block height")
	}
	return vrfMessage
}

func (f *fridayVRF) produce(nextBlockTime time.Time) time.Time {
	var chosenNumber int
	if f.node.status.GetHeight() != 0 {
		//getting VRFMessage by previous block body
		vrfMessage := f.getVRFMessage(f.node.status.GetHeight())

		//validate VRFMessage
		//bypass validate when produced genesis block
		if vrfMessage.PreviousBlockHeight != 0 {
			vrfErr := f.validateVRFMessage(vrfMessage)
			if vrfErr != nil {
				f.node.logger.Crit(vrfErr.Error())
				//TODO::replace to decide next action when invalid VRF situation
				panic(vrfErr)
			}
		}

		//calculate BP ID by VRF
		chosenNumber = f.calculateBPIDByVRF(vrfMessage)
	} else {
		//TODO::FIXME refectoring to initializeGenesisBlock
		//when firstly producing genesis block, cannot have previous block status
		chosenNumber = 0
	}

	// next := 0 if there is no completed block
	f.node.next = chosenNumber

	if f.node.next != f.node.id {
		// Not my turn
	} else {
		// My turn

		//Make VRFMessage
		var vrf vrfmessage.VRFMessage
		if f.node.status.GetHeight() != 0 {
			//make vrf by previous block
			vrf = f.makeVRFMessage(f.node.status.GetRecentBlock().Hash, f.node.status.GetHeight())
		} else {
			//TODO::FIXME refectoring to initializeGenesisBlock
			//for producing genesis block
			vrf = f.makeVRFMessage([32]byte{0}, 0)
		}

		// Produce new block
		newBlock := block.New(f.node.status.GetHeight()+1, nextBlockTime.UnixNano(), f.node.id, vrf)

		// Pre-prepare / send new block
		f.node.peer.sendBlock(newBlock)
		f.node.logger.Info("Block produced", "Height", newBlock.Header.Height, "Producer", newBlock.Header.Producer,
			"Timestmp", time.Unix(0, newBlock.Header.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash[:]))

	}

	return nextBlockTime.Add(f.node.parameter.blockTime)
}

func (f *fridayVRF) validateBlock(b block.Block) {
	// Validation
	if err := f.validate(b); err != nil {
		f.node.logger.Crit(err.Error())
		panic("There shoud be no byzitine nodes !")
		//return
	}

	f.node.logger.Info("Block received", "Blockheight", b.Header.Height)
	f.node.status.AppendBlock(b)

	// Prepare
	f.prepare(b)
	f.node.logger.Info("Block prepared", "Blockheight", b.Header.Height)

	// Commit / finalize
	f.finalize(b)
	f.node.logger.Info("Block finalized", "Blockheight", b.Header.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (f *fridayVRF) validate(b block.Block) error {
	// Validate producer
	if f.node.next != b.Header.Producer {
		return errors.New("cannot matched between f.node.next to block.Header.Producer")
	}

	// Validate block hash
	if b.Hash != block.CalculateHashFromBlock(b) {
		return errors.New("cannot invalid block hash")
	}

	return nil
}

func (f *fridayVRF) prepare(b block.Block) {
	// Generate dummy signature with -1
	sign := signature.New(f.node.id, signature.Prepare, b.Header.Height, -1)

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	f.node.pool.waitAndRemove(signature.Prepare, b.Header.Height, f.node.parameter.numValidators)
}

func (f *fridayVRF) finalize(b block.Block) {
	// Generate random signature
	sign := signature.New(f.node.id, signature.Commit, b.Header.Height, f.node.status.GetRandom())

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	signs := f.node.pool.waitAndRemove(signature.Commit, b.Header.Height, f.node.parameter.numValidators)

	// Finalize
	f.node.status.Finalize(b, signs)
}

func (f *fridayVRF) getRandomNumberFromSignatures(signs []signature.Signature) int {
	sum := 0
	for _, sign := range signs {
		if sign.Number < 0 {
			panic("Bad signature !")
		}
		sum += sign.Number
	}
	return sum % (f.node.parameter.numValidators)
}

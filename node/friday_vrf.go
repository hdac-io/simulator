package node

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/keytransparency/core/crypto/vrf"
	"github.com/google/keytransparency/core/crypto/vrf/p256"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/signature"
	"github.com/hdac-io/simulator/types"
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
			block := f.node.channel.readBlock()
			f.validateBlock(block)
		}
	} else {
		for {
			block := f.node.channel.readBlock()
			go f.validateBlock(block)
		}
	}
}

// VRF serialize and deserialize
func serialize(pkey vrf.PublicKey) []byte {
	pk := pkey.(*p256.PublicKey)
	return append(pk.PublicKey.X.Bytes(), pk.PublicKey.Y.Bytes()...)
}

func deserialize(data []byte) vrf.PublicKey {
	_, pkey := p256.GenerateKey()
	pk := pkey.(*p256.PublicKey)
	pk.X.SetBytes(data[:len(data)/2])
	pk.Y.SetBytes(data[len(data)/2:])

	return pk
}

func (f *fridayVRF) makeVRFMessage(blockHash [32]byte, height int) vrfmessage.VRFMessage {
	rand, proof := f.node.privKey.Evaluate(blockHash[:])
	message := vrfmessage.VRFMessage{
		Rand:                   rand,
		Proof:                  proof,
		PreviousProposerID:     f.node.id,
		PreviousProposerPubkey: serialize(f.node.pubKey),
		PreviousBlockHeight:    height,
	}
	return message
}

func (f *fridayVRF) validateVRFMessage(message vrfmessage.VRFMessage) error {

	if message.PreviousBlockHeight != f.node.status.GetHeight()-1 {
		return errors.New("Received previousBlockHeight is not equal then validator local height-1")
	}
	var targetHash [32]byte
	targetBlock, _ := f.node.status.GetBlock(message.PreviousBlockHeight)
	targetHash = targetBlock.Hash
	pubkey := deserialize(message.PreviousProposerPubkey)
	proofRand, err := pubkey.ProofToHash(
		targetHash[:],
		message.Proof)
	if proofRand != message.Rand || err != nil {
		return errors.New("Verify failed of received rand into vrfMessage")
	}

	return nil
}

func (f *fridayVRF) calculateBPIDByVRF(message vrfmessage.VRFMessage) int {
	// TODO::check overflow when based 32bit system
	so := int(binary.LittleEndian.Uint32(message.Rand[:]))
	chosenNumber := so%f.node.parameter.numValidators + 1
	f.node.logger.Debug("Calculate next validator", "Next validator", chosenNumber)

	return chosenNumber
}

func (f *fridayVRF) getVRFMessage(blockHeight int) vrfmessage.VRFMessage {
	var vrfMessage vrfmessage.VRFMessage

	// Getting VRFMessage by previous block body
	block, err := f.node.status.GetBlock(blockHeight)
	if err == nil {
		vrfMessage = block.VRF
	} else {
		panic("Block height index out of bound !")
	}
	return vrfMessage
}

func (f *fridayVRF) produce(nextBlockTime time.Time) time.Time {
	var chosenNumber int
	if f.node.status.GetHeight() != 0 {
		// Getting VRFMessage by previous block body
		vrfMessage := f.getVRFMessage(f.node.status.GetHeight())

		// Validate VRFMessage
		// Bypass validate when produced genesis block
		if vrfMessage.PreviousBlockHeight != 0 {
			vrfErr := f.validateVRFMessage(vrfMessage)
			if vrfErr != nil {
				f.node.logger.Crit(vrfErr.Error())
				// TODO::replace to decide next action when invalid VRF situation
				panic(vrfErr)
			}
		}

		// Calculate BP ID by VRF
		chosenNumber = f.calculateBPIDByVRF(vrfMessage)
	} else {
		// TODO::FIXME refectoring to initializeGenesisBlock
		// When firstly producing genesis block, cannot have previous block status
		chosenNumber = 1
	}

	// next := 0 if there is no completed block
	f.node.next = types.ID(chosenNumber)

	if f.node.next != f.node.id {
		// Not my turn
	} else {
		// My turn

		// Make VRFMessage
		var vrf vrfmessage.VRFMessage
		if f.node.status.GetHeight() != 0 {
			// Make vrf by previous block
			vrf = f.makeVRFMessage(f.node.status.GetRecentBlock().Hash, f.node.status.GetHeight())
		} else {
			// TODO::FIXME refectoring to initializeGenesisBlock
			// for producing genesis block
			vrf = f.makeVRFMessage([32]byte{0}, 0)
		}

		// Produce new block
		newBlock := block.New(f.node.status.GetHeight()+1, nextBlockTime.UnixNano(), f.node.id, vrf)

		// Pre-prepare / send new block
		f.node.channel.sendBlock(newBlock)
		f.node.logger.Info("Block produced", "Height", newBlock.Header.Height, "Producer", newBlock.Header.Producer,
			"Timestmp", time.Unix(0, newBlock.Header.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash[:]))

	}

	return nextBlockTime.Add(f.node.parameter.blockTime)
}

func (f *fridayVRF) validateBlock(b block.Block) {
	// Validation
	if err := f.validate(b); err != nil {
		f.node.logger.Crit(err.Error())
		panic("There shoud be no Byzantine nodes !")
		//return
	}

	f.node.logger.Info("Block received", "Height", b.Header.Height)
	f.node.status.AppendBlock(b)

	// Prepare
	f.prepare(b)
	f.node.logger.Info("Block prepared", "Height", b.Header.Height)

	// Commit / finalize
	f.finalize(b)
	f.node.logger.Info("Block finalized", "Height", b.Header.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (f *fridayVRF) validate(b block.Block) error {
	// FIXME: we should wait next validator calculation
	for f.node.next == 0 {
		time.Sleep(10 * time.Millisecond)

	}

	// Validate producer
	if f.node.next != b.Header.Producer {
		return errors.New("Invalid producer")
	}

	// Validate block hash
	if b.Hash != block.CalculateHashFromBlock(b) {
		return errors.New("Invalid block hash")
	}

	// FIXME: we should wait next validator calculation
	f.node.next = 0

	return nil
}

func (f *fridayVRF) prepare(b block.Block) {
	blsSign := f.node.blsSecretKey.SignHash(b.Hash[:])
	sign := signature.New(f.node.id, signature.Prepare, b.Header.Height, blsSign.Serialize())

	// Send piece to others
	f.node.channel.sendSignature(sign)

	// Collect signatures
	f.collectSignatures(signature.Prepare, b)
}

func (f *fridayVRF) finalize(b block.Block) {
	// Generate random signature
	blsSign := f.node.blsSecretKey.SignHash(b.Hash[:])
	sign := signature.New(f.node.id, signature.Commit, b.Header.Height, blsSign.Serialize())

	// Send piece to others
	f.node.channel.sendSignature(sign)

	// Collect signatures
	signs := f.collectSignatures(signature.Commit, b)

	// Finalize
	f.node.status.Finalize(b, signs)
}

func (f *fridayVRF) collectSignatures(kind signature.Kind, b block.Block) []signature.Signature {
	signs := f.node.pool.waitAndRemove(kind, b.Header.Height, f.node.parameter.numValidators)
	for _, s := range signs {
		id := s.ID
		// FIXME
		pubkey := bls.PublicKey{}
		pubkey.DeserializeHexStr(f.node.addressbook[id].PublicKey)

		blsSign := bls.Sign{}
		payload := s.Payload.([]byte)
		blsSign.Deserialize(payload)
		if !blsSign.VerifyHash(&pubkey, b.Hash[:]) {
			panic("There should be no Byzantine nodes !")
		}
	}

	return signs
}

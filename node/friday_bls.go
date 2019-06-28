package node

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/blsmessage"
	"github.com/hdac-io/simulator/signature"
)

type fridayBLS struct {
	node               *Node
	electionMessageMap map[int][]blsmessage.ValidatorMessage
	electionMutex      sync.RWMutex
}

func newFridayBLS(node *Node) consensus {
	return &fridayBLS{
		node:               node,
		electionMessageMap: make(map[int][]blsmessage.ValidatorMessage),
	}
}

func (f *fridayBLS) quorum() int {
	return f.node.parameter.numValidators*2/3 + 1
}

func (f *fridayBLS) start(genesisTime time.Time) {
	if f.node.parameter.numValidators <= f.quorum() {
		panic("validators less then quorum(2f+1)")
	}

	// Start producing loop
	go f.produceLoop(genesisTime)

	// Start validating loop
	go f.validationLoop()

	// Start election listening loop
	go f.electionListenLoop()
}

func (f *fridayBLS) produceLoop(genesisTime time.Time) {
	nextBlockTime := genesisTime
	for {
		time.Sleep(nextBlockTime.Sub(time.Now()))
		nextBlockTime = f.produce(nextBlockTime)
	}
}

func (f *fridayBLS) validationLoop() {
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

func (f *fridayBLS) electionListenLoop() {
	for {
		validatorMessage := f.node.peer.readValidatorMessage()
		messageHeight := validatorMessage.PreviousBlockHeight
		currentHeight := f.node.status.GetHeight()
		// TODO: remove keys under currentHeight (lower_bound)
		if currentHeight <= messageHeight {
			f.electionMutex.Lock()
			f.electionMessageMap[messageHeight] = append(f.electionMessageMap[messageHeight], validatorMessage)
			f.electionMutex.Unlock()

			f.node.logger.Info("received BLS-Validator Message",
				"message.validatorID", validatorMessage.ValidatorID,
				"message.PreviousHeight", messageHeight,
				"currentHeight", currentHeight)
		}
	}
}

func (f *fridayBLS) makeBLSMValidatorMessage(blockHash [32]byte, height int) blsmessage.ValidatorMessage {
	sign := f.node.blsPrivKey.SignHash(blockHash[:])
	if sign == nil {
		panic("failed bls signature signing")
	}
	message := blsmessage.NewValidatorMessage(f.node.id, *sign, f.node.blsPubKey, height)
	return message
}

func (f *fridayBLS) makeBLSMessage(blockHash [32]byte, height int) blsmessage.BLSMessage {
	sign := f.node.blsPrivKey.SignHash(blockHash[:])
	if sign == nil {
		panic("failed bls signature signing")
	}
	//TODO:: signing by validators
	message := blsmessage.New(*sign, f.node.blsPubKey, height)
	return message
}

func (f *fridayBLS) getBLSMessage(blockHeight int) blsmessage.BLSMessage {
	var blsMessage blsmessage.BLSMessage
	//getting BLSMessage by previous block body
	block, err := f.node.status.GetBlock(blockHeight)
	if err == nil {
		blsMessage = block.ElectionMessage.(blsmessage.BLSMessage)
	} else {
		panic("out-of-index block height")
	}
	return blsMessage
}

func (f *fridayBLS) validateBLSMessage(message blsmessage.BLSMessage) error {
	if message.PreviousBlockHeight != f.node.status.GetHeight()-1 {
		return errors.New("received previousBlockHeight is not equal then validator local height-1")
	}

	targetBlock, _ := f.node.status.GetBlock(message.PreviousBlockHeight)
	if !message.AggregatedElectionSign.VerifyHash(&message.AggregatedElectionPubkey, targetBlock.Hash[:]) {
		return errors.New("verify failed of received sign into blsMessage")

	}
	return nil
}

func (f *fridayBLS) calculateBPIDByBLS(message blsmessage.BLSMessage) int {
	//TODO::check overflow when based 32bit system
	so := int(binary.LittleEndian.Uint32(message.AggregatedElectionSign.Serialize()[:]))
	chosenNumber := so % f.node.parameter.numValidators
	f.node.logger.Debug("received bls-agrregatedSign to chosenNumber", "so", so, "chosenNumber", chosenNumber,
		"previous Block Height", message.PreviousBlockHeight)

	return chosenNumber
}

func (f *fridayBLS) produce(nextBlockTime time.Time) time.Time {

	var chosenNumber int
	if f.node.status.GetHeight() != 0 {
		//getting BLSMessage by previous block body
		blsMessage := f.getBLSMessage(f.node.status.GetHeight())

		//validate BLsMessage
		//bypass validate when produced genesis block
		if blsMessage.PreviousBlockHeight != 0 {
			blsErr := f.validateBLSMessage(blsMessage)
			if blsErr != nil {
				f.node.logger.Crit(blsErr.Error())
				//TODO::replace to decide next action when invalid BLS situation
				panic(blsErr)
			}
		}

		//calculate BP ID by BLS
		chosenNumber = f.calculateBPIDByBLS(blsMessage)
	} else {
		//TODO::FIXME refectoring to initializeGenesisBlock
		//when firstly producing genesis block, cannot have previous block status
		chosenNumber = 0
	}

	// next := 0 if there is no completed block
	f.node.next = chosenNumber

	if f.node.next != f.node.id {
	} else {
		// My turn
		var targetHash [32]byte
		if f.node.status.GetHeight() != 0 {
			//make bls by previous block
			targetHash = f.node.status.GetRecentBlock().Hash
		} else {
			//TODO::FIXME refectoring to initializeGenesisBlock
			//for producing genesis block
			targetHash = [32]byte{1, 3, 5, 7}
		}

		currentHeight := f.node.status.GetHeight()
		blsMessage := f.makeBLSMessage(targetHash, currentHeight)
		if currentHeight != 0 {
			f.electionMutex.RLock()
			voteCnt := len(f.electionMessageMap[currentHeight])
			f.node.logger.Info("successul election", "height", currentHeight, "vote count", voteCnt)

			if voteCnt >= f.quorum() {
				for _, validatorMessage := range f.electionMessageMap[currentHeight] {
					messageVerified := validatorMessage.Sign.VerifyHash(&validatorMessage.PublicKey, targetHash[:])
					if !messageVerified {
						//TODO::FIXME deceide next step when received fault validator mesage
						panic("verify failed received bls validator message")
					}

					blsMessage.AggregatedElectionSign.Add(&validatorMessage.Sign)
					blsMessage.AggregatedElectionPubkey.Add(&validatorMessage.PublicKey)
				}
			} else {
				//TODO::if not enoguh, waiting immediately before timeout
				panic("cannot received BLSValidatorMessage over then 2/3+1")
			}
			f.electionMutex.RUnlock()
		}
		if !blsMessage.AggregatedElectionSign.VerifyHash(&blsMessage.AggregatedElectionPubkey, targetHash[:]) {
			f.node.logger.Crit("verify failed aggregated bls sign")
		}

		// Produce new block
		newBlock := block.New(f.node.status.GetHeight()+1, nextBlockTime.UnixNano(), f.node.id, blsMessage)

		// Pre-prepare / send new block
		f.node.peer.sendBlock(newBlock)
		f.node.logger.Info("Block produced", "Height", newBlock.Header.Height, "Producer", newBlock.Header.Producer,
			"Timestmp", time.Unix(0, newBlock.Header.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash[:]))

	}

	return nextBlockTime.Add(f.node.parameter.blockTime)
}

func (f *fridayBLS) produceBLSValidatorMessage() {
	var targetHash [32]byte
	currentHeight := f.node.status.GetHeight()
	if currentHeight != 0 {
		//make bls by previous block
		targetHash = f.node.status.GetRecentBlock().Hash
	} else {
		//TODO::FIXME refectoring to initializeGenesisBlock
		//for producing genesis block
		targetHash = [32]byte{1, 3, 5, 7}
	}

	validatorMessage := f.makeBLSMValidatorMessage(targetHash, currentHeight)
	f.node.peer.sendValidatorMessage(validatorMessage)
}

func (f *fridayBLS) validateBlock(b block.Block) {
	// Validation
	if !f.validate(b) {
		panic("There shoud be no byzitine nodes !")
		//return
	}

	f.node.logger.Info("Block received", "Blockheight", b.Header.Height)
	f.node.status.AppendBlock(b)

	go f.produceBLSValidatorMessage()

	// Prepare
	f.prepare(b)
	f.node.logger.Info("Block prepared", "Blockheight", b.Header.Height)

	// Commit / finalize
	f.finalize(b)
	f.node.logger.Info("Block finalized", "Blockheight", b.Header.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (f *fridayBLS) validate(b block.Block) bool {
	// Validate producer
	if f.node.next != b.Header.Producer {
		return false
	}

	// Validate block hash
	if b.Hash != block.CalculateHashFromBlock(b) {
		return false
	}

	return true
}

func (f *fridayBLS) prepare(b block.Block) {
	// Generate dummy signature with -1
	sign := signature.New(f.node.id, signature.Prepare, b.Header.Height, -1)

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	f.node.pool.waitAndRemove(signature.Prepare, b.Header.Height, f.node.parameter.numValidators)
}

func (f *fridayBLS) finalize(b block.Block) {
	// Generate random signature
	sign := signature.New(f.node.id, signature.Commit, b.Header.Height, f.node.status.GetRandom())

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	signs := f.node.pool.waitAndRemove(signature.Commit, b.Header.Height, f.node.parameter.numValidators)

	// Finalize
	f.node.status.Finalize(b, signs)
}

func (f *fridayBLS) getRandomNumberFromSignatures(signs []signature.Signature) int {
	sum := 0
	for _, sign := range signs {
		if sign.Number < 0 {
			panic("Bad signature !")
		}
		sum += sign.Number
	}
	return sum % (f.node.parameter.numValidators)
}

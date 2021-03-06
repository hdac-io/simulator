package node

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/hdac-io/simulator/block"
	fridaytypes "github.com/hdac-io/simulator/types"
	"github.com/hdac-io/simulator/vrfmessage"
)

type fridayFBFT struct {
	node *Node
}

func newFridayFBFT(node *Node) consensus {
	return &fridayFBFT{node: node}
}

func (f *fridayFBFT) quorum() int {
	return f.node.parameter.numValidators*2/3 + 1
}

func (f *fridayFBFT) start(genesisTime time.Time) {
	if f.node.parameter.numValidators < f.quorum() {
		panic("number of validators less then quorum")
	}

	// Start producing loop
	go f.produceLoop(genesisTime)

	// Start validating loop
	go f.validationLoop()
}

func (f *fridayFBFT) produceLoop(genesisTime time.Time) {
	nextBlockTime := genesisTime
	for {
		time.Sleep(nextBlockTime.Sub(time.Now()))
		nextBlockTime = f.produce(nextBlockTime)
	}
}

func (f *fridayFBFT) validationLoop() {
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

func (f *fridayFBFT) getVRFMessage(blockHeight int) vrfmessage.VRFMessage {
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

func (f *fridayFBFT) getBlockProducerIDByHeight(height int) fridaytypes.ID {
	var chosenNumber fridaytypes.ID
	if height != 0 {
		//getting VRFMessage by previous block body
		vrfMessage := f.getVRFMessage(height)

		//validate VRFMessage
		//bypass validate when produced genesis block
		if vrfMessage.PreviousBlockHeight != 0 {
			targetBlock, _ := f.node.status.GetBlock(vrfMessage.PreviousBlockHeight)
			vrfErr := vrfMessage.Validate(targetBlock.Hash)
			if vrfErr != nil {
				f.node.logger.Crit(vrfErr.Error())
				//TODO::replace to decide next action when invalid VRF situation
				panic(vrfErr)
			}
		}

		//calculate BP ID by VRF
		chosenNumber = vrfMessage.CalculateBPID(f.node.parameter.numValidators)
	} else {
		//TODO::FIXME refectoring to initializeGenesisBlock
		//when firstly producing genesis block, cannot have previous block status
		chosenNumber = fridaytypes.ID(1)
	}

	return chosenNumber
}

func (f *fridayFBFT) produce(nextBlockTime time.Time) time.Time {

	chosenNumber := f.getBlockProducerIDByHeight(f.node.status.GetHeight())

	// next := 0 if there is no completed block
	f.node.next = fridaytypes.ID(chosenNumber)

	if f.node.next != f.node.id {
		// Not my turn
	} else {
		// My turn

		//Make VRFMessage
		var vrf vrfmessage.VRFMessage
		if f.node.status.GetHeight() != 0 {
			//make vrf by previous block
			vrf = vrfmessage.New(f.node.privKey, f.node.pubKey, f.node.id, f.node.status.GetRecentBlock().Hash, f.node.status.GetHeight())
		} else {
			//TODO::FIXME refectoring to initializeGenesisBlock
			//for producing genesis block
			vrf = vrfmessage.New(f.node.privKey, f.node.pubKey, f.node.id, [32]byte{0}, 0)
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

func (f *fridayFBFT) validateBlock(b block.Block) {
	f.node.logger.Info("Block received", "Blockheight", b.Header.Height)

	var isLeader bool
	//Check Current Leader or Validator
	currentLeaderID := f.getBlockProducerIDByHeight(f.node.status.GetHeight())
	if f.node.id == currentLeaderID {
		isLeader = true
	} else {
		isLeader = false
	}

	// Validation
	if err := f.validate(b); err != nil {
		f.node.logger.Crit(err.Error())
		panic("There shoud be no byzitine nodes !")
		//return
	}

	f.node.status.AppendBlock(b)

	if isLeader {
		f.fbftLeaderPhase(b)
	} else {
		f.fbftValidatorPhase(b)
	}
}

// FIXME: We assume that there is no byzantine nodes
func (f *fridayFBFT) validate(b block.Block) error {
	// FIXME: we should wait next validator calculation
	for f.node.next == 0 {
		time.Sleep(10 * time.Millisecond)

	}
	// Validate producer
	if f.node.next != b.Header.Producer {
		return errors.New("cannot matched between f.node.next to block.Header.Producer")
	}

	// Validate block hash
	if b.Hash != block.CalculateHashFromBlock(b) {
		return errors.New("cannot invalid block hash")
	}

	// FIXME: we should wait next validator calculation
	f.node.next = 0

	return nil
}

func (f *fridayFBFT) fbftLeaderPhase(b block.Block) {
	//Collecting prepare messages, Send 'PreparedMessagep
	preparedMessage, prepareErr := f.prepareLeaderPhase(b)
	if prepareErr != nil {
		panic(prepareErr.Error())
	}
	f.node.logger.Info("Block prepared", "Blockheight", b.Header.Height)

	//Collecting commit messages, Send 'CommitedMessage'
	finalizedSign, finalizedErr := f.finalizeLeaderPhase(b, preparedMessage)
	if finalizedErr != nil {
		// TODO::when failed finalization
		panic(finalizedErr.Error())
	}

	f.node.status.Finalize(b, finalizedSign)
	f.node.logger.Info("Block finalized", "Blockheight", b.Header.Height)

}

func (f *fridayFBFT) fbftValidatorPhase(b block.Block) {
	//Send 'PrepareMessage'
	prepareErr := f.prepareValidatorPhase(b)
	if prepareErr != nil {
		panic(prepareErr.Error())
	}

	//Handling to receive 'PreparedMessage'
	receivedPreparedMessage, preparedErr := f.onPreparedValidatorPhase(b)
	if preparedErr != nil {
		panic(prepareErr.Error())
	}
	f.node.logger.Info("Block prepared", "Blockheight", b.Header.Height)

	//Send 'CommitMessage'
	finalizeErr := f.finalizeValidatorPhase(b, receivedPreparedMessage)
	if finalizeErr != nil {
		panic(finalizeErr.Error())
	}

	//Handling to receive 'CommitedMessage'
	finalizedSign, finalizedErr := f.onFinalizedValidatorPhase(b, receivedPreparedMessage)
	if finalizedErr != nil {
		panic(finalizedErr.Error())

	}

	f.node.status.Finalize(b, finalizedSign)
	f.node.logger.Info("Block finalized", "Blockheight", b.Header.Height)
}

package node

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/signature"
)

type fridayBLS struct {
	node *Node
}

func newFridayBLS(node *Node) consensus {
	return &fridayBLS{node: node}
}

func (f *fridayBLS) start(genesisTime time.Time) {
	// Start producing loop
	go f.produceLoop(genesisTime)

	// Start validating loop
	go f.validationLoop()
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

func (f *fridayBLS) produce(nextBlockTime time.Time) time.Time {
	// Calculation
	signatures := f.node.status.GetRecentConfirmedSignature()
	chosenNumber := f.getRandomNumberFromSignatures(signatures)

	// next := 0 if there is no completed block
	f.node.next = chosenNumber

	if f.node.next != f.node.id {
		// Not my turn
	} else {
		// My turn

		// Produce new block
		newBlock := block.New(f.node.status.GetHeight()+1, nextBlockTime.UnixNano(), f.node.id)

		// Pre-prepare / send new block
		f.node.peer.sendBlock(newBlock)
		f.node.logger.Info("Block produced", "Height", newBlock.Height, "Producer", newBlock.Producer,
			"Timestmp", time.Unix(0, newBlock.Timestamp), "Hash", hex.EncodeToString(newBlock.Hash))

	}

	return nextBlockTime.Add(f.node.parameter.blockTime)
}

func (f *fridayBLS) validateBlock(b block.Block) {
	// Validation
	if !f.validate(b) {
		panic("There shoud be no byzitine nodes !")
		//return
	}

	f.node.logger.Info("Block received", "Blockheight", b.Height)
	f.node.status.AppendBlock(b)

	// Prepare
	f.prepare(b)
	f.node.logger.Info("Block prepared", "Blockheight", b.Height)

	// Commit / finalize
	f.finalize(b)
	f.node.logger.Info("Block finalized", "Blockheight", b.Height)
}

// FIXME: We assume that there is no byzantine nodes
func (f *fridayBLS) validate(b block.Block) bool {
	// Validate producer
	if f.node.next != b.Producer {
		return false
	}

	// Validate block hash
	if !bytes.Equal(b.Hash, block.CalculateHashFromBlock(b)) {
		return false
	}

	return true
}

func (f *fridayBLS) prepare(b block.Block) {
	// Generate dummy signature with -1
	sign := signature.New(f.node.id, signature.Prepare, b.Height, -1)

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	f.node.pool.waitAndRemove(signature.Prepare, b.Height, f.node.parameter.numValidators)
}

func (f *fridayBLS) finalize(b block.Block) {
	// Generate random signature
	sign := signature.New(f.node.id, signature.Commit, b.Height, f.node.status.GetRandom())

	// Send piece to others
	f.node.peer.sendSignature(sign)

	// Collect signatues
	signs := f.node.pool.waitAndRemove(signature.Commit, b.Height, f.node.parameter.numValidators)

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

package persistent

import (
	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/signature"
)

// Persistent represents persistent media
type Persistent struct {
	blocks     []block.Block
	signatures [][]signature.Signature
}

// New return inittial Persistent type
func New() Persistent {
	return Persistent{
		blocks:     make([]block.Block, 0),
		signatures: make([][]signature.Signature, 0),
	}
}

// AddBlock stores block
func (p *Persistent) AddBlock(block block.Block) {
	if len(p.blocks) != block.Height-1 {
		panic("Wrong block height !")
	}
	p.blocks = append(p.blocks, block)
}

// GetBlock retrieves block
func (p *Persistent) GetBlock(height int) block.Block {
	return p.blocks[height-1]
}

// AddSignature stores signature
func (p *Persistent) AddSignature(sign []signature.Signature) {
	if len(sign) == 0 || len(p.signatures) != sign[0].BlockHeight-1 {
		panic("Wrong block height !")
	}
	p.signatures = append(p.signatures, sign)
}

// GetSignature retrieves signature
func (p *Persistent) GetSignature(height int) []signature.Signature {
	if height < 1 {
		return []signature.Signature{}
	}
	return p.signatures[height-1]
}

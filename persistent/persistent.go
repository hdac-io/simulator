package persistent

import (
	"github.com/hdac-io/simulator/signature"
	"github.com/hdac-io/simulator/types"
)

// Persistent represents persistent media
type Persistent struct {
	blocks     []types.Block
	signatures [][]signature.Signature
}

// NewPersistent return inittial Persistent type
func NewPersistent() Persistent {
	return Persistent{
		blocks:     make([]types.Block, 0),
		signatures: make([][]signature.Signature, 0),
	}
}

// AddBlock stores block
func (p *Persistent) AddBlock(block types.Block) {
	if len(p.blocks) != block.Height-1 {
		panic("Wrong block height !")
	}
	p.blocks = append(p.blocks, block)
}

// GetBlock retrieves block
func (p *Persistent) GetBlock(height int) types.Block {
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

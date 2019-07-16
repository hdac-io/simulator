package status

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/persistent"
	"github.com/hdac-io/simulator/signature"
	log "github.com/inconshreveable/log15"
)

// Status represents network status
type Status struct {
	// Sync
	sync.RWMutex
	cond         *sync.Cond
	waitFinalize bool

	// Status
	finalizedHeight int
	confirmedHeight int
	height          int
	blocks          []block.Block

	// Persistent
	persistent persistent.Persistent

	// logger
	logger log.Logger
}

func randomSignature(max int) func() int {
	seed := int64(time.Now().Nanosecond())
	random := rand.New(rand.NewSource(seed))
	return func() int {
		return random.Intn(max)
	}
}

// New contstructs status
func New(id int64, max int, logger log.Logger) *Status {
	s := &Status{
		persistent: persistent.New(),
		logger:     logger,
	}
	s.cond = sync.NewCond(s)

	return s
}

// AppendBlock appends block
func (s *Status) AppendBlock(b block.Block) {
	s.Lock()
	// Increase height
	s.height++
	if s.height != b.Header.Height {
		panic("Block height mismatch !")
	}
	// Negative number and 0 mean there is no confirmed block
	s.confirmedHeight = s.height - (config.GetDefault().Consensus.LenULB + 1)
	// Append block
	s.blocks = append(s.blocks, b)
	s.Unlock()
}

// Finalize finalizing specified block
func (s *Status) Finalize(b block.Block, signs []signature.Signature) {
	s.Lock()
	for b.Header.Height > s.finalizedHeight+1 {
		s.logger.Warn("Previous block is not finalized yet !", "Current Finalizing height", b.Header.Height, "Previous finalized height", s.finalizedHeight)
		s.waitFinalize = true
		s.cond.Wait()
	}

	s.finalizedHeight = b.Header.Height
	s.blocks = s.blocks[1:]

	// Store finalized block
	s.persistent.AddBlock(b)
	// Store finalized signature
	s.persistent.AddSignature(signs)

	if s.waitFinalize {
		s.cond.Broadcast()
		s.waitFinalize = false
	}
	s.Unlock()
}

// GetHeight returns current block height
func (s *Status) GetHeight() int {
	return s.height
}

// GetBlock returns target block height
func (s *Status) GetBlock(height int) (block.Block, error) {
	if height <= s.finalizedHeight {
		return s.persistent.GetBlock(height), nil
	} else if height <= s.height {
		return s.blocks[height-s.finalizedHeight-1], nil
	} else {
		return block.Block{}, errors.New("out-of-index height")
	}
}

// GetRecentBlock returns recent block
func (s *Status) GetRecentBlock() block.Block {
	if len(s.blocks) > 0 {
		return s.blocks[len(s.blocks)-1]
	}
	return s.GetRecentFinalizedBlock()
}

// GetRecentFinalizedBlock returns recent finalized block
func (s *Status) GetRecentFinalizedBlock() block.Block {
	return s.persistent.GetBlock(s.finalizedHeight)
}

// GetRecentConfirmedBlock returns recent finalized block
func (s *Status) GetRecentConfirmedBlock() block.Block {
	return s.persistent.GetBlock(s.confirmedHeight)
}

// GetRecentConfirmedSignature returns recent finalized block
func (s *Status) GetRecentConfirmedSignature() []signature.Signature {
	return s.persistent.GetSignature(s.confirmedHeight)
}

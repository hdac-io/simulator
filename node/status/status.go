package status

import (
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
	GetRandom       func() int
	blocks          []block.Block

	// Persistent
	persistent persistent.Persistent

	// logger
	logger log.Logger
}

func randomSignature(unique int, max int) func() int {
	seed := int64(time.Now().Nanosecond() + unique)
	random := rand.New(rand.NewSource(seed))
	return func() int {
		return random.Intn(max)
	}
}

// New contstructs status
func New(id int, max int, logger log.Logger) *Status {
	s := &Status{
		GetRandom:  randomSignature(id, max),
		persistent: persistent.New(),
		logger:     logger,
	}
	s.cond = sync.NewCond(s)

	return s
}

// GetHeight returns current block height
func (s *Status) GetHeight() int {
	return s.height
}

// AppendBlock appends block
func (s *Status) AppendBlock(b block.Block) {
	s.Lock()
	// Increase height
	s.height++
	if s.height != b.Height {
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
	for b.Height > s.finalizedHeight+1 {
		s.logger.Warn("Previous block is not finalized yet !", "Current Finalizing height", b.Height, "Previous finalized height", s.finalizedHeight)
		s.waitFinalize = true
		s.cond.Wait()
	}

	s.finalizedHeight = b.Height
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

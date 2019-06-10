package fridayconsensus

import (
	"math/rand"
	"simulator/util"
	"sync"
	"time"
)

// Validator represents validator node
type Validator struct {
	id          int
	blockTime   time.Duration
	getRandom   func(int) int
	inbound     *Channel
	addressbook []*Channel
	blocks      []Block
}

func randomSignature(unique int) func(int) int {
	seed := int64(time.Now().Nanosecond() + unique)
	random := rand.New(rand.NewSource(seed))
	return func(max int) int {
		return random.Intn(max)
	}
}

// NewValidator construct validator
func NewValidator(id int, blockTime time.Duration) *Validator {
	v := Validator{
		id:        id,
		blockTime: blockTime,
		getRandom: randomSignature(id),
		inbound:   NewChannel(),
		blocks:    make([]Block, 0, 1024),
	}

	return &v
}

// GetAddress retrun validator's inbound address
func (v *Validator) GetAddress() *Channel {
	return v.inbound
}

// SetAddressbook sets validators` address
func (v *Validator) SetAddressbook(addressbook []*Channel) {
	v.addressbook = addressbook
}

// Start starts validator with genesis time
func (v *Validator) Start(genesisTime time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	// Add pre-genesis block
	v.blocks = append(v.blocks, Block{
		height: 0,
		// pre-genesis block timestamp is set "blockTime" before genesis time
		timestamp: genesisTime.Add(-(1 * time.Second)).UnixNano(),
	})

	for {
		recentBlock := &v.blocks[len(v.blocks)-1]

		// For genesis block and next one
		next := 0
		if recentBlock.height >= 2 {
			// calculation
			sum := 0
			for _, value := range recentBlock.signatures {
				sum += value
			}
			next = sum % (len(v.addressbook))
		}

		signatures := make([]int, len(v.addressbook))
		if recentBlock.height >= 1 {
			for index := range signatures {
				signatures[index] = v.inbound.readSignature()
			}
		}

		if next == v.id {
			// My turn
			now := time.Now()
			nextBlockTime := time.Unix(0, recentBlock.timestamp).Add(v.blockTime)
			time.Sleep(nextBlockTime.Sub(now))

			// Produce new block
			newBlock := Block{
				height:     recentBlock.height + 1,
				timestamp:  nextBlockTime.UnixNano(),
				producer:   v.id,
				signatures: signatures,
			}

			// Send new block
			for _, peer := range v.addressbook {
				peer.sendBlock(newBlock)
			}
			util.Log("#", v.id, "Block produced\n", newBlock)
		} else {
			// Not my turn
		}
		block := v.inbound.readBlock()
		v.receiveBlock(block)
	}
}

func (v *Validator) stop() {
	// Clean validator up
}

func (v *Validator) receiveBlock(b Block) {
	// Validation
	if !v.validate() {
		return
	}

	// pre-commit
	v.preCommit()

	// commit
	v.commit(b)

	util.Log("#", v.id, "Block received and committed. Blockheight =", b.height)
}

func (v *Validator) preCommit() {
	// generate random signature
	sig := v.getRandom(len(v.addressbook))

	// send piece to others
	for _, peer := range v.addressbook {
		peer.sendSignature(sig)
	}
}

func (v *Validator) commit(b Block) {
	v.blocks = append(v.blocks, b)
}

func (v *Validator) validate() bool {
	// FIXME: We assume that there is no byzantine nodes
	return true
}
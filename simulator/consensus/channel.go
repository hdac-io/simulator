package consensus

import (
	"time"
)

type Channel struct {
	block     chan Block
	signature chan int
}

func NewChannel() *Channel {
	n := Channel{
		block: make(chan Block, 1),
		// FIXME: We should make P2P channel
		signature: make(chan int, 3),
	}

	return &n
}

func (c *Channel) sendSignature(sig int) {
	go func() {
		networkDelay()
		c.signature <- sig
	}()
}

func (c *Channel) sendBlock(block Block) {
	go func() {
		networkDelay()
		c.block <- block
	}()
}

func (c *Channel) readSignature() int {
	return <-c.signature
}

func (c *Channel) readBlock() Block {
	return <-c.block
}

func networkDelay() {
	time.Sleep(200 * time.Millisecond)
}

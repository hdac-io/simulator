package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

// BlockHeader seperated Body
type BlockHeader struct {
	Height    int
	Timestamp int64
	Producer  int
}

//  LeaderElectionMessage interface included VRF, BLS
type LeaderElectionMessage = interface{}

// Block represents simple block structure
type Block struct {
	Header          BlockHeader
	Hash            [32]byte
	ElectionMessage LeaderElectionMessage
}

// New constructs block
func New(height int, timestamp int64, producer int, electionMessage LeaderElectionMessage) Block {
	b := Block{
		Header: BlockHeader{
			Height:    height,
			Timestamp: timestamp,
			Producer:  producer,
		},
		ElectionMessage: electionMessage,
	}
	b.Hash = CalculateHashFromBlock(b)

	return b
}

// CalculateHashFromBlock returns calculated hash using block contents
func CalculateHashFromBlock(b Block) [32]byte {
	//TODO:: decide encode function(ex: ethereum-rlp, ...)
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(b.Header)

	return sha256.Sum256(buf.Bytes())
}

package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"

	"github.com/hdac-io/simulator/vrfmessage"
)

// BlockHeader seperated Body
type BlockHeader struct {
	Height    int
	Timestamp int64
	Producer  int
}

// Block represents simple block structure
type Block struct {
	Header BlockHeader
	Hash   [32]byte
	VRF    vrfmessage.VRFMessage
}

// New constructs block
func New(height int, timestamp int64, producer int, vrf vrfmessage.VRFMessage) Block {
	b := Block{
		Header: BlockHeader{
			Height:    height,
			Timestamp: timestamp,
			Producer:  producer,
		},
		VRF: vrf,
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

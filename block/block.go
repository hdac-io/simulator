package block

import (
	"crypto/sha256"
	"encoding/binary"
)

// Block represents simple block structure
type Block struct {
	Hash      []byte
	Height    int
	Timestamp int64
	Producer  int
}

// New constructs block
func New(height int, timestamp int64, producer int) Block {
	b := Block{
		Height:    height,
		Timestamp: timestamp,
		Producer:  producer,
	}
	b.Hash = CalculateHashFromBlock(b)

	return b
}

// CalculateHashFromBlock returns calculated hash using block contents
func CalculateHashFromBlock(b Block) []byte {
	hash := sha256.New()
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, uint64(b.Height))
	hash.Write(buffer)
	buffer = make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, uint64(b.Timestamp))
	hash.Write(buffer)
	buffer = make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, uint64(b.Producer))
	hash.Write(buffer)

	return hash.Sum(nil)
}

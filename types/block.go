package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// Block represents simple block structure
type Block struct {
	Hash         []byte
	Height       int
	Timestamp    int64
	Producer     int
	ChosenNumber int
}

// NewBlock constructs block
func NewBlock(height int, timestamp int64, producer int, chosenNumber int) Block {
	b := Block{
		Height:       height,
		Timestamp:    timestamp,
		Producer:     producer,
		ChosenNumber: chosenNumber,
	}
	b.Hash = hashFromBlock(b)

	return b
}

func hashFromBlock(b Block) []byte {
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
	buffer = make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, uint64(b.ChosenNumber))
	hash.Write(buffer)

	return hash.Sum(nil)
}

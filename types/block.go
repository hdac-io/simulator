package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type BlockHeader struct {
	Height       int
	Timestamp    int64
	Producer     int
	ChosenNumber int
}

// Block represents simple block structure
type Block struct {
	Header BlockHeader
	Hash   [32]byte
}

func NewBlock(height int, timestamp int64, producer int, chosenNumber int) Block {
	var block = Block{}
	blockHeader := BlockHeader{
		Height:       height,
		Timestamp:    timestamp,
		Producer:     producer,
		ChosenNumber: chosenNumber,
	}
	block.Header = blockHeader

	//TODO:: decide encode function(ex: ethereum-rlp, ...)
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(block.Header)

	block.Hash = sha256.Sum256(buf.Bytes())

	return block
}

type Signature struct {
	Id          int
	BlockHeight int
	Number      int
}

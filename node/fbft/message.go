package fbft

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"

	"github.com/hdac-io/simulator/bls"
)

//Send by Leader

//PreparedMessage is aggregated result by received PrepareMessages
type PreparedMessage struct {
	AggregatedSign   bls.Sign
	AggregatedPubKey bls.PublicKey
}

//Hash receiver method is message to sha256 hash
func (message *PreparedMessage) Hash() [32]byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(message)

	return sha256.Sum256(buf.Bytes())
}

//CommitedMessage is aggregated result by received CommitMessage
type CommitedMessage struct {
	AggregatedSign   bls.Sign
	AggregatedPubKey bls.PublicKey
}

//Send by Validators

//PrepareMessage is sign for received block
type PrepareMessage struct {
	Sign      bls.Sign
	PublicKey bls.PublicKey
}

//CommitMessage is sign for received PreparedMessage
type CommitMessage struct {
	Sign      bls.Sign
	PublicKey bls.PublicKey
}

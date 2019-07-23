package fbft

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"

	"github.com/hdac-io/simulator/bls"
)

func mashalSignAndPubkey(sign bls.Sign, key bls.PublicKey) []byte {
	mashaledJSON, _ := json.Marshal([]string{sign.SerializeToHexStr(), key.SerializeToHexStr()})
	return mashaledJSON
}
func unmashalSignAndPubkey(payload []byte) (sign bls.Sign, key bls.PublicKey, err error) {

	var serializedHexStrs []string
	err = json.Unmarshal(payload, &serializedHexStrs)
	if err == nil {
		err = sign.DeserializeHexStr(serializedHexStrs[0])
	}
	if err == nil {
		err = key.DeserializeHexStr(serializedHexStrs[1])
	}

	return sign, key, err
}

// Message used for Prepare, Prepared, Commit, Commited
type Message struct {
	Sign   bls.Sign
	Pubkey bls.PublicKey
}

//Hash receiver method is message to sha256 hash
func (message *Message) Hash() [32]byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(message.Serialize())

	return sha256.Sum256(buf.Bytes())
}

// Serialize return mashared json message
func (message *Message) Serialize() []byte {
	return mashalSignAndPubkey(message.Sign, message.Pubkey)
}

// Deserialize return unmashared json message
func (message *Message) Deserialize(payload []byte) error {
	var err error
	message.Sign, message.Pubkey, err = unmashalSignAndPubkey(payload)
	return err
}

package vrfmessage

import (
	"encoding/binary"
	"errors"

	"github.com/google/keytransparency/core/crypto/vrf"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/hdac-io/simulator/types"
)

// VRFMessage contains VRF validation informations
type VRFMessage struct {
	Rand                   [32]byte
	Proof                  []byte
	PreviousProposerID     types.ID
	PreviousProposerPubkey []byte
	PreviousBlockHeight    int
}

// VRF serialize
func serialize(pkey vrf.PublicKey) []byte {
	pk := pkey.(*p256.PublicKey)
	return append(pk.PublicKey.X.Bytes(), pk.PublicKey.Y.Bytes()...)
}

// VRF deserialize
func deserialize(data []byte) vrf.PublicKey {
	_, pkey := p256.GenerateKey()
	pk := pkey.(*p256.PublicKey)
	pk.X.SetBytes(data[:len(data)/2])
	pk.Y.SetBytes(data[len(data)/2:])

	return pk
}

// New is return VRFMessage
func New(privKey vrf.PrivateKey, pubKey vrf.PublicKey, proposerID types.ID, blockHash [32]byte, height int) VRFMessage {
	rand, proof := privKey.Evaluate(blockHash[:])
	message := VRFMessage{
		Rand:                   rand,
		Proof:                  proof,
		PreviousProposerID:     proposerID,
		PreviousProposerPubkey: serialize(pubKey),
		PreviousBlockHeight:    height,
	}
	return message
}

// Validate is return validate result of message
func (message *VRFMessage) Validate(targetHash [32]byte) error {
	pubkey := deserialize(message.PreviousProposerPubkey)
	proofRand, err := pubkey.ProofToHash(
		targetHash[:],
		message.Proof)
	if proofRand != message.Rand || err != nil {
		return errors.New("Verify failed of received rand into vrfMessage")
	}

	return nil
}

// CalculateBPID is return to extract validator ID
func (message *VRFMessage) CalculateBPID(numValidators int) types.ID {
	// TODO::check overflow when based 32bit system
	so := int(binary.LittleEndian.Uint32(message.Rand[:]))
	chosenNumber := so%numValidators + 1

	return types.ID(chosenNumber)
}

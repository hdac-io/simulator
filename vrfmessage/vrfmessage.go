package vrfmessage

import (
	"github.com/google/keytransparency/core/crypto/vrf"
)

type VRFMessage struct {
	Rand                   [32]byte
	Proof                  []byte
	PreviousProposerID     int
	PreviousProposerPubkey vrf.PublicKey
	PreviousBlockHeight    int
}

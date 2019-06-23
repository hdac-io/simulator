package fridayconsensus

import (
	"github.com/google/keytransparency/core/crypto/vrf"
)

type vrfMessage struct {
	rand                   [32]byte
	proof                  []byte
	previousProposerID     int
	previousProposerPubkey vrf.PublicKey
	previousBlockHeight    int
}

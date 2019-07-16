package vrfmessage

import "github.com/hdac-io/simulator/types"

// VRFMessage contains VRF validation informations
type VRFMessage struct {
	Rand                   [32]byte
	Proof                  []byte
	PreviousProposerID     types.ID
	PreviousProposerPubkey []byte
	PreviousBlockHeight    int
}

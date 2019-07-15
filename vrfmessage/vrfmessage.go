package vrfmessage

// VRFMessage contains VRF validation informations
type VRFMessage struct {
	Rand                   [32]byte
	Proof                  []byte
	PreviousProposerID     int
	PreviousProposerPubkey []byte
	PreviousBlockHeight    int
}

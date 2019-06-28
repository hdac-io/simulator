package blsmessage

import (
	"github.com/hdac-io/simulator/blsmessage/bls"
)

type ValidatorMessage struct {
	ValidatorID         int
	Sign                bls.Sign
	PublicKey           bls.PublicKey
	PreviousBlockHeight int
}

func NewValidatorMessage(validatorID int, validatorSign bls.Sign, validatorPubKey bls.PublicKey, previousBlockHeight int) ValidatorMessage {
	return ValidatorMessage{
		ValidatorID:         validatorID,
		Sign:                validatorSign,
		PublicKey:           validatorPubKey,
		PreviousBlockHeight: previousBlockHeight,
	}
}

type BLSMessage struct {
	AggregatedElectionSign   bls.Sign
	AggregatedElectionPubkey bls.PublicKey
	PreviousBlockHeight      int
}

func New(leaderSign bls.Sign, leaderPubkey bls.PublicKey, previousBlockHeight int) BLSMessage {
	return BLSMessage{
		AggregatedElectionSign:   leaderSign,
		AggregatedElectionPubkey: leaderPubkey,
		PreviousBlockHeight:      previousBlockHeight,
	}
}

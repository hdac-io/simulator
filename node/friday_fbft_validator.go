package node

import (
	"errors"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/node/fbft"
	"github.com/hdac-io/simulator/signature"
)

func (f *fridayFBFT) prepareValidatorPhase(b block.Block) error {
	//Prepare Phase - validate received block(announce), send prepare message
	blockSign := f.node.blsSecretKey.SignHash(b.Hash[:])
	if blockSign == nil {
		return errors.New("Failed block bls signing")
	}

	toSendPrepareMessage := fbft.Message{
		Sign:   *blockSign,
		Pubkey: *f.node.blsSecretKey.GetPublicKey(),
	}
	serializedMessage := toSendPrepareMessage.Serialize()

	prepareTx := signature.New(f.node.id, signature.Prepare, b.Header.Height, serializedMessage)
	f.node.channel.sendSignature(prepareTx)
	f.node.logger.Info("Send prepare messsage", "blockheight", b.Header.Height)

	return nil
}

func (f *fridayFBFT) onPreparedValidatorPhase(b block.Block) (fbft.Message, error) {
	//OnPrepared Phase - wait leader bls-aggregated message
	receivedTx := f.node.pool.waitAndRemove(signature.Prepared, b.Header.Height, 1)
	if len(receivedTx) != 1 {
		return fbft.Message{}, errors.New("Cannot received leader prepared message")
	}

	// TODO:: Add more leader message validate condition
	// - check between known leader public key to received leader public key
	// - check match between received public keys to known validator public keys
	var deserializedMessage fbft.Message
	err := deserializedMessage.Deserialize(receivedTx[0].Payload.([]byte))
	if err != nil {
		return fbft.Message{}, err
	}

	if !deserializedMessage.Sign.VerifyHash(&deserializedMessage.Pubkey, b.Hash[:]) {
		return fbft.Message{}, errors.New("Invalid Leader prepared message")
	}
	f.node.logger.Info("Received prepared leader message", "blockheight", b.Header.Height)
	return deserializedMessage, nil
}

func (f *fridayFBFT) finalizeValidatorPhase(b block.Block, preparedMessage fbft.Message) error {
	//Commit Phase - send commit message
	messageHash := preparedMessage.Hash()
	messageSign := f.node.blsSecretKey.SignHash(messageHash[:])
	if messageSign == nil {
		return errors.New("failed message bls signing")
	}

	toSendMessage := fbft.Message{
		Sign:   *messageSign,
		Pubkey: *f.node.blsSecretKey.GetPublicKey(),
	}
	commitTx := signature.New(f.node.id, signature.Commit, b.Header.Height, toSendMessage.Serialize())
	f.node.channel.sendSignature(commitTx)
	f.node.logger.Info("send prepare messsage", "blockheight", b.Header.Height)

	return nil
}

func (f *fridayFBFT) onFinalizedValidatorPhase(b block.Block, preparedMessage fbft.Message) ([]signature.Signature, error) {
	//OnCommited Phase -  Wait leader bls-aggregated message
	receivedTx := f.node.pool.waitAndRemove(signature.Commited, b.Header.Height, 1)
	if len(receivedTx) != 1 {
		return []signature.Signature{}, errors.New("Cannot received leader commited message")
	}

	// TODO:: Add more leader message validate condition
	// - check between known leader public key to received leader public key
	// - check match between received public keys to known validator public keys
	messageHash := preparedMessage.Hash()
	var deserializedMessage fbft.Message
	err := deserializedMessage.Deserialize(receivedTx[0].Payload.([]byte))
	if err != nil {
		return []signature.Signature{}, err
	}

	if !deserializedMessage.Sign.VerifyHash(&deserializedMessage.Pubkey, messageHash[:]) {
		return []signature.Signature{}, errors.New("Invalid aggregated-bls on leader commited message")
	}
	f.node.logger.Info("Received commited leader message", "blockheight", b.Header.Height)
	return receivedTx, nil
}

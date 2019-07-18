package node

import (
	"errors"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/node/fbft"
	"github.com/hdac-io/simulator/signature"
)

func (f *fridayFBFT) prepareLeaderPhase(b block.Block) (fbft.Message, error) {
	f.node.logger.Debug("Enter prepareLeaderPhase", "blockHeight", b.Header.Height)

	//Prepare Phase
	toSendMessage := fbft.Message{}

	collectStartTime := time.Now()
	// TODO::handling when timeout situation
	receivedSignTxs := f.node.pool.waitAndRemove(signature.Prepare, b.Header.Height, f.quorum())
	elpasedReceiveTime := time.Since(collectStartTime)
	if len(receivedSignTxs) < f.quorum() {
		return fbft.Message{}, errors.New("Cannot receive prepare messages more than quorum")
	}

	f.node.logger.Debug("Received prepare Txs over than quorum", "blockHeight", b.Header.Height, "elpasedReceiveTime", elpasedReceiveTime.String())

	aggregationStartTime := time.Now()
	for _, signTx := range receivedSignTxs {
		if signTx.Kind != signature.Prepare {
			return fbft.Message{}, errors.New("Cannot matched Tx kind")
		}

		var deserializedMessage fbft.Message
		err := deserializedMessage.Deserialize(signTx.Payload.([]byte))
		if err != nil {
			return fbft.Message{}, err
		}
		if !deserializedMessage.Sign.VerifyHash(&deserializedMessage.Pubkey, b.Hash[:]) {
			// TODO::handling when received invalidate prepare message
			return fbft.Message{}, errors.New("Cannot verified validator prepare message")
		}

		toSendMessage.Sign.Add(&deserializedMessage.Sign)
		toSendMessage.Pubkey.Add(&deserializedMessage.Pubkey)
	}
	elapsedAggregationTime := time.Since(aggregationStartTime)

	preparedLeaderTx := signature.New(f.node.id, signature.Prepared, b.Header.Height, toSendMessage.Serialize())
	f.node.channel.sendSignature(preparedLeaderTx)
	f.node.logger.Debug("Success BLS-Aggregation of Prepare Messages", "blockHeight", b.Header.Height, "elapsedAggregationTime", elapsedAggregationTime.String())
	return toSendMessage, nil
}

func (f *fridayFBFT) finalizeLeaderPhase(b block.Block, preparedMessage fbft.Message) ([]signature.Signature, error) {
	f.node.logger.Debug("Enter finalizeLeaderPhase", "blockHeight", b.Header.Height)
	//Commit Phase
	toSendMessage := fbft.Message{}

	collectStartTime := time.Now()
	// TODO::handling when timeout situation
	receivedSignTxs := f.node.pool.waitAndRemove(signature.Commit, b.Header.Height, f.quorum())
	elpasedReceiveTime := time.Since(collectStartTime)
	if len(receivedSignTxs) < f.quorum() {
		return []signature.Signature{}, errors.New("Cannot receive prepare messages more than quorum")
	}

	f.node.logger.Debug("Received commit Txs over then quorum", "blockHeight", b.Header.Height, "elpasedReceiveTime", elpasedReceiveTime.String())

	aggregationStartTime := time.Now()
	for _, signTx := range receivedSignTxs {
		if signTx.Kind != signature.Commit {
			return []signature.Signature{}, errors.New("Cannot matched Tx kind")
		}

		var deserializedMessage fbft.Message
		err := deserializedMessage.Deserialize(signTx.Payload.([]byte))
		if err != nil {
			return []signature.Signature{}, err
		}

		compareHash := preparedMessage.Hash()
		if !deserializedMessage.Sign.VerifyHash(&deserializedMessage.Pubkey, compareHash[:]) {
			// TODO::handling when received invalid commit message
			return []signature.Signature{}, errors.New("Invalid validator commit message")
		}

		toSendMessage.Sign.Add(&deserializedMessage.Sign)
		toSendMessage.Pubkey.Add(&deserializedMessage.Pubkey)
	}
	elapsedAggregationTime := time.Since(aggregationStartTime)

	commitedLeaderTx := signature.New(f.node.id, signature.Commited, b.Header.Height, toSendMessage.Serialize())
	f.node.channel.sendSignature(commitedLeaderTx)
	f.node.logger.Debug("Success BLS-Aggregation of commit messages", "blockHeight", b.Header.Height, "elapsedAggregationTime", elapsedAggregationTime.String())

	return []signature.Signature{commitedLeaderTx}, nil
}

package node

import (
	"errors"
	"time"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/node/fbft"
	"github.com/hdac-io/simulator/signature"
)

func (f *fridayFBFT) prepareLeaderPhase(b block.Block) (fbft.PreparedMessage, error) {
	//Prepare Phase
	toSendMessage := fbft.PreparedMessage{}

	collectStartTime := time.Now()
	// TODO::handling when timeout situation
	receivedSignTxs := f.node.pool.waitAndRemove(signature.Prepare, b.Header.Height, f.quorum())
	elpasedReceiveTime := time.Since(collectStartTime)
	if len(receivedSignTxs) < f.quorum() {
		return fbft.PreparedMessage{}, errors.New("Cannot receive prepare messages more than quorum")
	}

	f.node.logger.Debug("Received prepare Txs over than quorum", "blockHeight", b.Header.Height, "elpasedReceiveTime", elpasedReceiveTime.String())

	aggregationStartTime := time.Now()
	for _, signTx := range receivedSignTxs {
		if signTx.Kind != signature.Prepare {
			return fbft.PreparedMessage{}, errors.New("Cannot matched Tx kind")
		}

		prepareMessage := signTx.Payload.(fbft.PrepareMessage)
		if !prepareMessage.Sign.VerifyHash(&prepareMessage.PublicKey, b.Hash[:]) {
			// TODO::handling when received invalidate prepare message
			return fbft.PreparedMessage{}, errors.New("Cannot verified validator prepare message")
		}

		toSendMessage.AggregatedSign.Add(&prepareMessage.Sign)
		toSendMessage.AggregatedPubKey.Add(&prepareMessage.PublicKey)
	}
	elapsedAggregationTime := time.Since(aggregationStartTime)

	preparedLeaderTx := signature.New(f.node.id, signature.Prepared, b.Header.Height, toSendMessage)
	f.node.peer.sendSignature(preparedLeaderTx)
	f.node.logger.Debug("Success BLS-Aggregation of Prepare Messages", "blockHeight", b.Header.Height, "elapsedAggregationTime", elapsedAggregationTime.String())
	return toSendMessage, nil
}

func (f *fridayFBFT) finalizeLeaderPhase(b block.Block, preparedMessage fbft.PreparedMessage) ([]signature.Signature, error) {
	//Commit Phase
	toSendMessage := fbft.CommitedMessage{}

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

		compareHash := preparedMessage.Hash()
		commitMessage := signTx.Payload.(fbft.CommitMessage)
		if !commitMessage.Sign.VerifyHash(&commitMessage.PublicKey, compareHash[:]) {
			// TODO::handling when received invalid commit message
			return []signature.Signature{}, errors.New("Invalid validator commit message")
		}

		toSendMessage.AggregatedSign.Add(&commitMessage.Sign)
		toSendMessage.AggregatedPubKey.Add(&commitMessage.PublicKey)
	}
	elapsedAggregationTime := time.Since(aggregationStartTime)

	commitedLeaderTx := signature.New(f.node.id, signature.Commited, b.Header.Height, toSendMessage)
	f.node.peer.sendSignature(commitedLeaderTx)
	f.node.logger.Debug("Success BLS-Aggregation of commit messages", "blockHeight", b.Header.Height, "elapsedAggregationTime", elapsedAggregationTime.String())

	return []signature.Signature{commitedLeaderTx}, nil
}

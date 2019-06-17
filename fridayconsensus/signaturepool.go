package fridayconsensus

type signaturepool map[int][]signature

func newSignaturePool() signaturepool {
	return make(signaturepool)
}

func (s signaturepool) add(sig signature) {
	signatures := s[sig.blockHeight]
	signatures = append(signatures, sig)
	s[sig.blockHeight] = signatures
}

func (s signaturepool) remove(blockHeight int) []signature {
	signatures := s[blockHeight]
	delete(s, blockHeight)
	return signatures
}

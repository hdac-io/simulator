package fridayconsensus

import "sync"

type notifiableSignature struct {
	sync.Mutex
	cond       *sync.Cond
	target     int
	signatures []signature
}

type signaturepool struct {
	sync.Mutex
	signatures map[int]*notifiableSignature
}

func newSignaturePool() *signaturepool {
	return &signaturepool{
		signatures: make(map[int]*notifiableSignature),
	}
}

func (s *signaturepool) get(height int) *notifiableSignature {
	s.Lock()
	sig, exists := s.signatures[height]
	if !exists {
		sig = &notifiableSignature{}
		sig.cond = sync.NewCond(sig)

		s.signatures[height] = sig
	}
	s.Unlock()

	return sig
}

func (s *signaturepool) wait(height int, number int) {
	sig := s.get(height)
	sig.Lock()
	sig.target = number
	if sig.target > 0 && sig.target != len(sig.signatures) {
		sig.cond.Wait()
	}
	sig.Unlock()
}

func (s *signaturepool) add(newSig signature) {
	sig := s.get(newSig.blockHeight)
	sig.Lock()
	sig.signatures = append(sig.signatures, newSig)
	sig.Unlock()

	if sig.target > 0 && sig.target == len(sig.signatures) {
		sig.cond.Signal()
	}
}

func (s *signaturepool) remove(height int) []signature {
	sig := s.get(height)
	delete(s.signatures, height)
	return sig.signatures
}

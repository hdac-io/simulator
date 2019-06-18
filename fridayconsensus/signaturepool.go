package fridayconsensus

import "sync"

type notifiableSignature struct {
	sync.Mutex
	cond       *sync.Cond
	target     int
	signatures []signature
}

type signaturepool struct {
	sync.RWMutex
	signatures map[int]*notifiableSignature
}

func newSignaturePool() *signaturepool {
	return &signaturepool{
		signatures: make(map[int]*notifiableSignature),
	}
}

func (s *signaturepool) get(height int) *notifiableSignature {
	s.RLock()
	sig, exists := s.signatures[height]
	s.RUnlock()
	if !exists {
		s.Lock()
		// Check again under locked condition
		sig, exists = s.signatures[height]
		if !exists {
			sig = &notifiableSignature{}
			sig.target = -1
			sig.cond = sync.NewCond(sig)

			s.signatures[height] = sig
		}
		s.Unlock()
	}

	return sig
}

func (s *signaturepool) waitAndRemove(height int, number int) []signature {
	sig := s.get(height)
	sig.Lock()
	if sig.target != -1 {
		panic("Must not enter here !")
	}
	sig.target = number
	if sig.target > 0 && sig.target != len(sig.signatures) {
		sig.cond.Wait()
	}
	s.Lock()
	delete(s.signatures, height)
	s.Unlock()
	sig.Unlock()

	return sig.signatures
}

func (s *signaturepool) add(newSig signature) {
	sig := s.get(newSig.blockHeight)
	sig.Lock()
	sig.signatures = append(sig.signatures, newSig)

	if sig.target > 0 && sig.target == len(sig.signatures) {
		sig.cond.Signal()
	}
	sig.Unlock()
}

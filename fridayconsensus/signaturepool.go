package fridayconsensus

import (
	"sync"
)

type notifiableSignature struct {
	sync.Mutex
	cond       *sync.Cond
	target     int
	signatures []signature
}

type signatureMap map[int]*notifiableSignature

type signaturepool struct {
	sync.RWMutex
	signatures [numKind]signatureMap
}

func newSignaturePool() *signaturepool {
	return &signaturepool{
		signatures: [numKind]signatureMap{make(signatureMap), make(signatureMap)},
	}
}

func (s *signaturepool) get(kind kind, height int) *notifiableSignature {
	s.RLock()
	sig, exists := s.signatures[kind][height]
	s.RUnlock()
	if !exists {
		s.Lock()
		// Check again under locked condition
		sig, exists = s.signatures[kind][height]
		if !exists {
			sig = &notifiableSignature{}
			sig.target = -1
			sig.cond = sync.NewCond(sig)

			s.signatures[kind][height] = sig
		}
		s.Unlock()
	}

	return sig
}

func (s *signaturepool) waitAndRemove(kind kind, height int, number int) []signature {
	sig := s.get(kind, height)
	sig.Lock()
	if sig.target != -1 {
		panic("Must not enter here !")
	}
	sig.target = number
	for sig.target > 0 && sig.target != len(sig.signatures) {
		sig.cond.Wait()
	}
	s.Lock()
	delete(s.signatures[kind], height)
	s.Unlock()
	sig.Unlock()

	return sig.signatures
}

func (s *signaturepool) add(kind kind, newSign signature) {
	sign := s.get(kind, newSign.blockHeight)
	sign.Lock()
	sign.signatures = append(sign.signatures, newSign)

	if sign.target > 0 && sign.target == len(sign.signatures) {
		sign.cond.Signal()
	}
	sign.Unlock()
}

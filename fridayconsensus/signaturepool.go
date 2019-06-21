package fridayconsensus

import (
	"sync"

	"github.com/hdac-io/simulator/signature"
)

type notifiableSignature struct {
	sync.Mutex
	cond       *sync.Cond
	target     int
	signatures []signature.Signature
}

type signatureMap map[int]*notifiableSignature

type signaturepool struct {
	sync.RWMutex
	signatures [signature.NumKind]signatureMap
}

func newSignaturePool() *signaturepool {
	return &signaturepool{
		signatures: [signature.NumKind]signatureMap{make(signatureMap), make(signatureMap)},
	}
}

func (s *signaturepool) get(kind signature.Kind, height int) *notifiableSignature {
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

func (s *signaturepool) waitAndRemove(kind signature.Kind, height int, number int) []signature.Signature {
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

func (s *signaturepool) add(kind signature.Kind, newSign signature.Signature) {
	sign := s.get(kind, newSign.BlockHeight)
	sign.Lock()
	sign.signatures = append(sign.signatures, newSign)

	if sign.target > 0 && sign.target == len(sign.signatures) {
		sign.cond.Signal()
	}
	sign.Unlock()
}

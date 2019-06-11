package fridayconsensus

import "github.com/hdac-io/simulator/types"

func newSignature(id int, height int, number int) types.Signature {
	return types.Signature{
		Id:          id,
		BlockHeight: height,
		Number:      number,
	}
}

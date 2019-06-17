package fridayconsensus

type signature struct {
	id          int
	blockHeight int
	number      int
}

func newSignature(id int, height int, number int) signature {
	return signature{
		id:          id,
		blockHeight: height,
		number:      number,
	}
}

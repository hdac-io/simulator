package fridayconsensus

type kind int

const (
	prepare kind = 0
	commit  kind = 1
)

const numKind = 2

type signature struct {
	id          int
	kind        kind
	blockHeight int
	number      int
}

func newSignature(id int, kind kind, height int, number int) signature {
	return signature{
		id:          id,
		kind:        kind,
		blockHeight: height,
		number:      number,
	}
}

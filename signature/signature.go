package signature

// Kind is signature enum type
type Kind int

// Signature kind
const (
	Prepare Kind = 0
	Commit  Kind = 1
)

// NumKind is number of signatures kind
const NumKind = 2

// Signature represents validation signature
type Signature struct {
	ID          int
	Kind        Kind
	BlockHeight int
	Number      int
}

// New returns signature type
func New(id int, kind Kind, height int, number int) Signature {
	return Signature{
		ID:          id,
		Kind:        kind,
		BlockHeight: height,
		Number:      number,
	}
}

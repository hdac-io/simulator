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

// Payload type for Various Kinds
type Payload interface{}

// Signature represents validation signature
type Signature struct {
	ID          int
	Kind        Kind
	BlockHeight int
	Payload     Payload
}

// New returns signature type
func New(id int, kind Kind, height int, payload Payload) Signature {
	return Signature{
		ID:          id,
		Kind:        kind,
		BlockHeight: height,
		Payload:     payload,
	}
}

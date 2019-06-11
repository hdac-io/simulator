package types

// Block represents simple block structure
type Block struct {
	Height       int
	Timestamp    int64
	Producer     int
	ChosenNumber int
}

type Signature struct {
	Id          int
	BlockHeight int
	Number      int
}

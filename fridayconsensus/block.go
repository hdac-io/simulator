package fridayconsensus

import (
	"fmt"
	"time"
)

// Block represents simple block structure
type block struct {
	height       int
	timestamp    int64
	producer     int
	chosenNumber int
}

func (b block) String() string {
	return fmt.Sprintf("Height : %d\nProducer: %d\nBlock time: %s\nChosen number: %d", b.height, b.producer, time.Unix(0, b.timestamp), b.chosenNumber)
}

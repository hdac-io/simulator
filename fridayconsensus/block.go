package fridayconsensus

import (
	"fmt"
	"time"
)

// Block represents simple block structure
type Block struct {
	height     int
	timestamp  int64
	producer   int
	signatures []int
}

func (b Block) String() string {
	return fmt.Sprintf("Height : %d\nProducer: %d\nBlock time: %s", b.height, b.producer, time.Unix(0, b.timestamp))
}

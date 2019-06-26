package node

import "time"

type consensus interface {
	start(genesisTime time.Time)
}

package consensus

import (
	"github.com/hdac-io/simulator/consensus/friday"
)

// Consensus is interface of various consensus
type Consensus interface {
	Start()
}

// Get retreives specified consensus
func Get(name string) Consensus {
	// TODO: reflection
	if name == "friday" {
		return friday.Friday{}
	}

	panic("Consensus is not found !")
}

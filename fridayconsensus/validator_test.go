package fridayconsensus

import (
	"os"
	"testing"

	simulcfg "github.com/hdac-io/simulator/config"
	"github.com/hdac-io/simulator/network"
	"github.com/stretchr/testify/require"
)

var cfg *simulcfg.Config

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	cfg = simulcfg.GetDefault()
	os.Exit(m.Run())
}

func TestNewValidator(t *testing.T) {
	inputValidatorID := 0
	inputBlockTime := cfg.Consensus.BlockTime
	validator := NewValidator(inputValidatorID, inputBlockTime, 1, cfg.Consensus.LenULB)

	require.Equal(t, inputValidatorID, validator.id)
	require.Equal(t, 0, len(validator.addressbook))
}

func TestValidatorInitialization(t *testing.T) {
	validator := NewValidator(0, cfg.Consensus.BlockTime, 1, cfg.Consensus.LenULB)
	require.True(t, validator.initialize([]*network.Network{validator.GetAddress()}))

	require.Equal(t, 1, len(validator.addressbook))
	require.Equal(t, 1, len(validator.peer.outbound))
}

//TODO:: to separate and Wrap one functionality inside logic.
// func TestSingleBlockProduceAndReceiveBetweenTwoNodes(t *testing.T) {

// 	//initialize validators and addressbooks
// 	validators := make([]*Validator, 2)
// 	addressbook := make([]*network.Network, 2)
// 	for id := range validators {
// 		validators[id] = NewValidator(id, cfg.Consensus.BlockTime, 2, cfg.Consensus.LenULB)
// 		addressbook[id] = validators[id].GetAddress()
// 	}

// 	for id := range validators {
// 		validators[id].initialize(addressbook)
// 		go validators[id].receiveLoop()
// 	}

// 	//produce
// 	genesisTime := time.Now().Add(1 * time.Second).Round(1 * time.Second)
// 	validators[0].produce(genesisTime)

// 	//receive
// 	//TODO:: Need to wrapping another method of calling reedBlock and receiveBlock.
// 	receivedBlockBy0 := validators[0].peer.readBlock()
// 	receivedBlockBy1 := validators[1].peer.readBlock()

// 	validateCompleteChan := make(chan bool)

// 	var wg sync.WaitGroup
// 	wg.Add(2)
// 	go func() {
// 		wg.Wait()
// 		validateCompleteChan <- true
// 	}()
// 	go func() {
// 		validators[0].validateBlock(receivedBlockBy0)
// 		wg.Done()
// 	}()
// 	go func() {
// 		validators[1].validateBlock(receivedBlockBy1)
// 		wg.Done()
// 	}()
// 	for {
// 		select {
// 		case <-validateCompleteChan:
// 			//comapre between Producer to validator
// 			require.Equal(t, validators[0].id, receivedBlockBy1.Producer)
// 			require.Equal(t, receivedBlockBy0, receivedBlockBy1)
// 			return

// 		case <-time.After(time.Second * 2):
// 			t.Fatal("cannot finalized block")
// 			return
// 		}

// 	}
// }

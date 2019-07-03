package node

import (
	"math/rand"
	"sync"
	"time"

	"github.com/google/keytransparency/core/crypto/vrf"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/node/status"
	"github.com/hdac-io/simulator/persistent"
	log "github.com/inconshreveable/log15"
)

// Node represents validator node
type Node struct {
	// Chain parameters
	parameter parameter

	// Consensus
	consensus consensus

	// Validator data
	id        int
	validator bool
	next      int

	// Peer-to-peer network
	peer *channel

	// Status
	status *status.Status

	// Transaction pool
	pool *signaturepool

	// Persistent
	persistent persistent.Persistent

	// Logger
	logger log.Logger

	//VRF Key Pair
	privKey vrf.PrivateKey
	pubKey  vrf.PublicKey

	//BLS Key Pair
	blsSecretKey bls.SecretKey
	blsPublicKey bls.PublicKey
}

type parameter struct {
	numValidators int
	lenULB        int
	blockTime     time.Duration
}

func randomSignature(unique int, max int) func() int {
	seed := int64(time.Now().Nanosecond() + unique)
	random := rand.New(rand.NewSource(seed))
	return func() int {
		return random.Intn(max)
	}
}

// New constructs node
func New(id int, numValidators int, lenULB int) *Node {
	parameter := parameter{
		numValidators: numValidators,
		lenULB:        lenULB,
	}

	n := &Node{
		id:         id,
		peer:       newChannel(),
		parameter:  parameter,
		persistent: persistent.New(),
		pool:       newSignaturePool(),
		logger:     log.New("Validator", id),
	}
	n.status = status.New(id, numValidators, n.logger)
	// FIXME: configurable
	n.consensus = newFridayFBFT(n)

	// Initailze VRF Key Pair
	n.privKey, n.pubKey = p256.GenerateKey()

	n.blsSecretKey.SetByCSPRNG()
	n.blsPublicKey = *n.blsSecretKey.GetPublicKey()
	n.logger.Info("created secret key", "private", n.blsSecretKey.SerializeToHexStr(), "public", n.blsPublicKey.SerializeToHexStr())

	return n
}

// NewValidator constructs validator node
func NewValidator(id int, numValidators int, lenULB int, blockTime time.Duration) *Node {
	n := New(id, numValidators, lenULB)
	n.validator = true
	n.parameter.blockTime = blockTime

	return n
}

// GetAddress returns validator's inbound address
func (n *Node) GetAddress() *network.Network {
	return n.peer.inbound.network
}

func (n *Node) initialize(addressbook []*network.Network) bool {
	// Start channel
	n.peer.start(addressbook)

	//FIXME: more detail successful
	return true
}

// Start starts validator with genesis time
func (n *Node) Start(genesisTime time.Time, addressbook []*network.Network, wg *sync.WaitGroup) {
	defer wg.Done()

	if !n.initialize(addressbook) {
		panic("Initialization failed !")
	}

	// Wait for genesis
	time.Sleep(genesisTime.Sub(time.Now()))

	n.consensus.start(genesisTime)

	// Start receiving loop
	n.receiveLoop()
}

func (n *Node) receiveLoop() {
	for {
		signature := n.peer.readSignature()
		n.pool.add(signature.Kind, signature)
	}
}

func (n *Node) stop() {
	// Clean validator up
}

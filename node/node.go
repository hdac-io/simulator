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

// Identity represents validator identity
type Identity struct {
	Address   network.Address
	PublicKey *bls.PublicKey
}

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

	identities []Identity

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

	sec bls.SecretKey

	//VRF Key Pair
	privKey vrf.PrivateKey
	pubKey  vrf.PublicKey

	// BLS secret
	blsSecretKey bls.SecretKey
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
	n.consensus = newFridayVRF(n)

	return n
}

// NewValidator constructs validator node
func NewValidator(id int, numValidators int, lenULB int, blockTime time.Duration) *Node {
	n := New(id, numValidators, lenULB)
	n.validator = true
	n.parameter.blockTime = blockTime

	// Initailze VRF key pair
	n.privKey, n.pubKey = p256.GenerateKey()

	// Initialize BLS secret
	n.blsSecretKey.SetByCSPRNG()

	return n
}

// GetIdentity returns validator's inbound address and public key
func (n *Node) GetIdentity() Identity {
	return Identity{
		Address:   n.peer.address,
		PublicKey: n.blsSecretKey.GetPublicKey(),
	}
}

func (n *Node) initialize(identities []Identity) bool {
	// Start channel

	// FIXME
	// Node has higher ID connect to nodes have lower ID
	id := 0
	for id <= n.id {
		n.peer.addPeer(identities[id].Address)
		id++
	}

	n.identities = identities

	return true
}

// Start starts validator with genesis time
func (n *Node) Start(genesisTime time.Time, identities []Identity, wg *sync.WaitGroup) {
	defer wg.Done()

	if !n.initialize(identities) {
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

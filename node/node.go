package node

import (
	"sync"
	"time"

	"github.com/google/keytransparency/core/crypto/vrf"
	"github.com/google/keytransparency/core/crypto/vrf/p256"
	"github.com/hdac-io/simulator/bls"
	"github.com/hdac-io/simulator/node/status"
	"github.com/hdac-io/simulator/persistent"
	"github.com/hdac-io/simulator/types"
	log "github.com/inconshreveable/log15"
)

// Node represents validator node
type Node struct {
	// Chain parameters
	parameter parameter

	// Consensus
	consensus consensus

	// Validator data
	id        types.ID
	validator bool
	next      types.ID

	// Address book
	addressbook Addressbook

	// Peer-to-channel network
	channel *channel

	// Status
	status *status.Status

	// Transaction pool
	pool *signaturepool

	// Persistent
	persistent persistent.Persistent

	// Logger
	logger log.Logger

	// VRF Key Pair
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

// New constructs node
func New(id types.ID, addressbook Addressbook, lenULB int) *Node {
	parameter := parameter{
		numValidators: len(addressbook),
		lenULB:        lenULB,
	}

	n := &Node{
		id:          id,
		addressbook: addressbook,
		channel:     newChannel(addressbook[id]),
		parameter:   parameter,
		persistent:  persistent.New(),
		pool:        newSignaturePool(),
		logger:      log.New("Validator", id),
	}
	n.status = status.New(int64(id), len(addressbook), n.logger)
	// FIXME: configurable
	n.consensus = newFridayVRF(n)

	return n
}

// NewValidator constructs validator node
func NewValidator(id types.ID, addressbook Addressbook, lenULB int, blockTime time.Duration) *Node {
	n := New(id, addressbook, lenULB)
	n.validator = true
	n.parameter.blockTime = blockTime

	// Initailze VRF key pair
	n.logger.Info("Initialize VRF key")
	n.privKey, n.pubKey = p256.GenerateKey()

	// Initialize BLS secret
	n.logger.Info("Initialize BLS key")
	n.blsSecretKey.DeserializeHexStr(addressbook[id].Secret)

	return n
}

func (n *Node) prepare() bool {
	// Add known peers
	n.channel.addKnownPeers(n.addressbook)

	return true
}

// Start starts validator with genesis time
func (n *Node) Start(genesisTime time.Time, wg *sync.WaitGroup) {
	defer wg.Done()

	// Prepare peer-to-peer network, 4 seconds before genesis time
	time.Sleep(genesisTime.Add(-9 * time.Second).Sub(time.Now()))
	if !n.prepare() {
		panic("Initialization failed !")
	}

	// Wait for genesis time
	time.Sleep(genesisTime.Sub(time.Now()))

	n.consensus.start(genesisTime)

	// Start receiving loop
	n.receiveLoop()
}

func (n *Node) receiveLoop() {
	for {
		signature := n.channel.readSignature()
		n.pool.add(signature.Kind, signature)
	}
}

func (n *Node) stop() {
	// Clean validator up
}

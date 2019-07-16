package net

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicSingleReadWrite(t *testing.T) {
	net := NewNetwork()
	net.getDelay = func() time.Duration {
		return time.Millisecond
	}

	net.Write(1)
	readed := net.Read()
	require.Equal(t, readed, 1)
}

func TestNetworkDelayTimeSimulation(t *testing.T) {
	net := NewNetwork()

	testDelaySecond := 2
	net.getDelay = func() time.Duration {
		return time.Second * time.Duration(testDelaySecond)
	}

	//nonblock
	net.Write(1)

	startTime := time.Now()
	net.Read()
	elpasedTime := time.Since(startTime).Seconds()

	require.True(t, elpasedTime >= float64(testDelaySecond))
}

func TestWriteOverflow(t *testing.T) {
	net := NewNetwork()
	net.getDelay = func() time.Duration {
		return time.Millisecond
	}

	for i := 0; i <= 1024; i++ {
		net.Write(i)
	}

	//wait asyncronous write
	for len(net.network) < 1024 {
		time.Sleep(time.Millisecond * 10)
	}

	//boom!
	networkChanIsFull := false
	select {
	case net.network <- 2: // Put 2 in the channel unless it is full
	default:
		networkChanIsFull = true
	}

	require.True(t, networkChanIsFull)
}

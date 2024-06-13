package test_libp2p

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var protocolID = protocol.ID("/test")

func Test(t *testing.T) {
	closers := make([]io.Closer, 0, 3)
	defer func() {
		for _, closer := range closers {
			require.NoError(t, closer.Close())
		}
	}()

	net := mocknet.New()
	closers = append(closers, net)

	// create advertiser
	advertiser, dhtAdvertiser := createPeer(t, net)
	closers = append(closers, advertiser, dhtAdvertiser)

	// create peer 1
	peer1, dhtPeer1 := createPeer(t, net)
	closers = append(closers, peer1, dhtPeer1)

	// connect peer 1 to advertiser
	connectPeers(t, net, peer1, advertiser)

	// create peer 1
	peer2, dhtPeer2 := createPeer(t, net)
	closers = append(closers, peer2, dhtPeer2)

	// connect peer 2 to advertiser
	connectPeers(t, net, peer2, advertiser)

	// link peer1 and peer 2
	_, err := net.LinkPeers(peer1.ID(), peer2.ID())
	require.NoError(t, err)

	// bootstrap all
	err = dhtAdvertiser.Bootstrap(context.TODO())
	require.NoError(t, err)

	err = dhtPeer1.Bootstrap(context.TODO())
	require.NoError(t, err)

	err = dhtPeer2.Bootstrap(context.TODO())
	require.NoError(t, err)

	// wait for bootstrap
	time.Sleep(time.Second * 3)

	assert.Equal(t, 3, len(advertiser.Peerstore().Peers()))
	assert.Equal(t, 3, len(peer1.Peerstore().Peers()))
	assert.Equal(t, 3, len(peer2.Peerstore().Peers()))
}

func createPeer(t *testing.T, net mocknet.Mocknet) (host.Host, *dht.IpfsDHT) {
	h, err := net.GenPeer()
	require.NoError(t, err)

	dhtPeer, err := dht.New(
		context.TODO(),
		h,
		dht.ProtocolPrefix(protocolID),
		dht.RoutingTableRefreshPeriod(time.Second),
		dht.Mode(dht.ModeServer),
	)
	require.NoError(t, err)

	return h, dhtPeer
}

func connectPeers(
	t *testing.T,
	net mocknet.Mocknet,
	h1 host.Host,
	h2 host.Host,
) {
	_, err := net.LinkPeers(h1.ID(), h2.ID())
	require.NoError(t, err)

	err = h1.Connect(context.TODO(), getConnectableAddr(t, h2))
	require.NoError(t, err)
}

func getConnectableAddr(t *testing.T, h host.Host) peer.AddrInfo {
	addr, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p/%s", h.Addrs()[0].String(), h.ID().String()))
	require.NoError(t, err)

	return *addr
}

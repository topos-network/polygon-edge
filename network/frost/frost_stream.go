package frost

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	manet "github.com/multiformats/go-multiaddr/net"
)

type FrostStream struct {
	ctx      context.Context
	streamCh chan network.Stream
}

func NewFrostStream() *FrostStream {
	return &FrostStream{
		ctx:      context.Background(),
		streamCh: make(chan network.Stream),
	}
}

type Context struct {
	context.Context
	PeerID peer.ID
}

func (s *FrostStream) Client(stream network.Stream) *network.Stream {
	return &stream
}

func (s *FrostStream) Handler() func(network.Stream) {
	fmt.Println(">>>>>>>>>>>>>>>  FROST Handler called:")

	return func(stream network.Stream) {
		go func() {
			for {
				b := <-s.streamCh
				fmt.Println(">>>>>>>>>>>>>>> FROST RECEIVED BYTES >> :", b)
			}
		}()

		select {
		case <-s.ctx.Done():
			return
		case s.streamCh <- stream:
		}
	}
}

// --- listener ---

func (s *FrostStream) Accept() (net.Conn, error) {
	select {
	case <-s.ctx.Done():
		return nil, io.EOF
	case stream := <-s.streamCh:
		return &streamConn{Stream: stream}, nil
	}
}

// Addr implements the net.Listener interface
func (s *FrostStream) Addr() net.Addr {
	return fakeLocalAddr()
}

func (s *FrostStream) Close() error {
	return nil
}

// streamConn represents a net.Conn wrapped to be compatible with net.conn
type streamConn struct {
	network.Stream
}

type wrapLibp2pAddr struct {
	id peer.ID
	net.Addr
}

// LocalAddr returns the local address.
func (c *streamConn) LocalAddr() net.Addr {
	addr, err := manet.ToNetAddr(c.Stream.Conn().LocalMultiaddr())
	if err != nil {
		return fakeRemoteAddr()
	}

	return &wrapLibp2pAddr{Addr: addr, id: c.Stream.Conn().LocalPeer()}
}

// RemoteAddr returns the remote address.
func (c *streamConn) RemoteAddr() net.Addr {
	addr, err := manet.ToNetAddr(c.Stream.Conn().RemoteMultiaddr())
	if err != nil {
		return fakeRemoteAddr()
	}

	return &wrapLibp2pAddr{Addr: addr, id: c.Stream.Conn().RemotePeer()}
}

var _ net.Conn = &streamConn{}

// fakeLocalAddr returns a dummy local address.
func fakeLocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}
}

// fakeRemoteAddr returns a dummy remote address.
func fakeRemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.1.0.1"),
		Port: 0,
	}
}

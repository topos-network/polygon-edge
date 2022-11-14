package frost

import (
	proto "github.com/0xPolygon/polygon-edge/consensus/ibft/frost/messages"
)

// Transport defines an interface
// the node uses to communicate with other peers
type Transport interface {
	// Multicast multicasts the message to other peers
	MulticastFrost(message *proto.Message)
}

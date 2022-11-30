package frost

import (
	"errors"
	"fmt"

	"github.com/0xPolygon/polygon-edge/network/event"
	"github.com/hashicorp/go-hclog"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const PeerID = "peerID"

var (
	ErrInvalidChainID   = errors.New("invalid chain ID")
	ErrNoAvailableSlots = errors.New("no available Slots")
)

// networkingServer defines the base communication interface between
// any networking server implementation and the FrostService
type networkingServer interface {

	// NewIdentityClient returns strem connection
	NewFrostClient(peerID peer.ID) (network.Stream, error)

	// DisconnectFromPeer attempts to disconnect from the specified peer
	DisconnectFromPeer(peerID peer.ID, reason string)

	// AddPeer adds a peer to the networking server's peer store
	AddFrostPeer(id peer.ID, direction network.Direction)

	// UpdatePendingConnCount updates the pendingPeerConnections connection count for the direction [Thread safe]
	UpdateFrostPendingConnCount(delta int64, direction network.Direction)

	// EmitEvent emits the specified peer event on the base networking server
	EmitEvent(event *event.PeerEvent)

	// CONNECTION INFORMATION //
	HasFreeFrostConnectionSlot(direction network.Direction) bool

	AddFrostPendingPeer(peerID peer.ID, direction network.Direction)

	RemoveFrostPendingPeer(peerID peer.ID)

	HasPendingStatus(peerID peer.ID) bool

	HasFrostPendingStatus(peerID peer.ID) bool
}

// FrostService is a networking service used to handle peer handshaking.
// It acts as a gatekeeper to peer connectivity
type FrostService struct {
	logger     hclog.Logger     // The FrostService logger
	baseServer networkingServer // The interface towards the base networking server

	chainID int64   // The chain ID of the network
	hostID  peer.ID // The base networking server's host peer ID
}

// NewFrostService returns a new instance of the FrostService
func NewFrostService(
	server networkingServer,
	logger hclog.Logger,
	chainID int64,
	hostID peer.ID,
) *FrostService {
	return &FrostService{
		logger:     logger.Named("frost"),
		baseServer: server,
		chainID:    chainID,
		hostID:     hostID,
	}
}

func (f *FrostService) GetNotifyBundle() *network.NotifyBundle {
	return &network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			peerID := conn.RemotePeer()
			f.logger.Debug("Checking connection for FROST:", "peer", peerID, "direction", conn.Stat().Direction)

			if f.baseServer.HasFrostPendingStatus(peerID) {
				// handshake has already started
				return
			}

			if !f.baseServer.HasFreeFrostConnectionSlot(conn.Stat().Direction) && !f.baseServer.HasPendingStatus(peerID) {
				f.disconnectFromPeer(peerID, ErrNoAvailableSlots.Error())

				return
			}

			// Mark the peer as pending (pending handshake)
			f.baseServer.AddFrostPeer(peerID, conn.Stat().Direction)

			go func() {
				eventType := event.PeerDialCompleted

				if err := f.handleConnected(peerID, conn.Stat().Direction); err != nil {
					f.baseServer.RemoveFrostPendingPeer(peerID)
					// Close the connection to the peer
					if !f.baseServer.HasPendingStatus(peerID) {
						f.disconnectFromPeer(peerID, err.Error())
					}

					eventType = event.PeerFailedToConnect
				} else {
					// Mark the peer as no longer pending
					f.baseServer.RemoveFrostPendingPeer(peerID)
				}

				// Emit an adequate event
				f.baseServer.EmitEvent(&event.PeerEvent{
					PeerID: peerID,
					Type:   eventType,
				})
			}()
		},
	}
}

// hasPendingStatus checks if a peer is pending handshake [Thread safe]
// func (i *FrostService) hasPendingStatus(id peer.ID) bool {
// 	_, ok := i.pendingPeerConnections.Load(id)

// 	return ok
// }

// // removePendingStatus removes the pending status from a peer,
// // and updates adequate counter information [Thread safe]
// func (i *FrostService) removePendingStatus(peerID peer.ID) {
// 	if value, loaded := i.pendingPeerConnections.LoadAndDelete(peerID); loaded {
// 		direction, ok := value.(network.Direction)
// 		if !ok {
// 			return
// 		}

// 		i.baseServer.UpdatePendingFrostConnCount(-1, direction)
// 	}
// }

// // addPendingStatus adds the pending status to a peer,
// // and updates adequate counter information [Thread safe]
// func (i *FrostService) addPendingStatus(id peer.ID, direction network.Direction) {
// 	if _, loaded := i.pendingPeerConnections.LoadOrStore(id, direction); !loaded {
// 		i.baseServer.UpdatePendingFrostConnCount(1, direction)
// 	}
// }

// disconnectFromPeer disconnects from the specified peer
func (f *FrostService) disconnectFromPeer(peerID peer.ID, reason string) {
	f.baseServer.DisconnectFromPeer(peerID, reason)
}

// handleConnected handles new network connections (handshakes)
func (f *FrostService) handleConnected(peerID peer.ID, direction network.Direction) error {
	fmt.Println(">>>>>>>>>>>>> FROST handle connected peer id: ", peerID)
	_, clientErr := f.baseServer.NewFrostClient(peerID)
	if clientErr != nil {
		return fmt.Errorf(
			"unable to create new frost client connection, %w",
			clientErr,
		)
	}

	fmt.Println(">>>>>>>>>>>>> FROST Here we should communicate and check frost info of the peer: ", peerID)

	// Validate that the peers are working on the same chain
	// if status.Chain != resp.Chain {
	// 	return ErrInvalidChainID
	// }

	// If this is a NOT temporary connection, save it
	f.baseServer.AddFrostPeer(peerID, direction)

	return nil
}

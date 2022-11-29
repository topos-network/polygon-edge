package network

import (
	"github.com/0xPolygon/polygon-edge/network/common"
	peerEvent "github.com/0xPolygon/polygon-edge/network/event"
	"github.com/0xPolygon/polygon-edge/network/frost"
	"github.com/0xPolygon/polygon-edge/network/identity"
	"github.com/armon/go-metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// NewFrostClient returns a new frost stream client connection
func (s *Server) NewFrostClient(peerID peer.ID) (network.Stream, error) {
	// Create a new stream connection and return it
	stream, err := s.NewStream(common.Frost, peerID)
	if err != nil {
		return nil, err
	}

	// Identity protocol connections are temporary and not saved anywhere
	return stream, nil
}

// AddFrostPeer adds a new topos node peer to the networking server's peer list,
// and updates relevant counters and metrics
func (s *Server) AddFrostPeer(id peer.ID, direction network.Direction) {
	s.logger.Info("Frost peer connected", "id", id.String())

	// Update the peer connection info
	if connectionExists := s.addFrostPeerInfo(id, direction); connectionExists {
		// The peer connection information was already present in the networking
		// server, so no connection metrics should be updated further
		return
	}

	// Emit the event alerting listeners
	s.emitEvent(id, peerEvent.FrostPeerConnected)
}

// addFrostPeerInfo updates the networking server's internal frost peer info table
// and returns a flag indicating if the same peer connection previously existed.
// In case the peer connection previously existed, this is a noop
func (s *Server) addFrostPeerInfo(id peer.ID, direction network.Direction) bool {
	s.frostPeersLock.Lock()
	defer s.frostPeersLock.Unlock()

	frostConnectionInfo, frostConnectionExists := s.frostPeers[id]
	if frostConnectionExists && frostConnectionInfo.connDirections[direction] {
		// Check if this peer already has an active connection status (saved info).
		// There is no need to do further processing
		return true
	}

	// Check if the connection info is already initialized
	if !frostConnectionExists {
		// Create a new record for the connection info
		frostConnectionInfo = &FrostPeerConnInfo{
			Info:           s.host.Peerstore().PeerInfo(id),
			connDirections: make(map[network.Direction]bool),
		}
	}

	// Save the connection info to the networking server
	frostConnectionInfo.connDirections[direction] = true

	s.frostPeers[id] = frostConnectionInfo

	// Update connection counters
	s.frostConnectionCounts.UpdateConnCountByDirection(1, direction)
	s.updateFrostConnCountMetrics(direction)

	// Update the metric stats
	metrics.SetGauge([]string{"frost_peers"}, float32(len(s.peers)))

	return false
}

// UpdatePendingConnCount updates the pending connection count in the specified direction [Thread safe]
func (s *Server) UpdateFrostPendingConnCount(delta int64, direction network.Direction) {
	s.frostConnectionCounts.UpdatePendingConnCountByDirection(delta, direction)

	s.updateFrostPendingConnCountMetrics(direction)
}

// setupFrost sets up the identity service for the node
func (s *Server) setupFrost() error {
	// Create an instance of the identity service
	identityService := identity.NewIdentityService(
		s,
		s.logger,
		int64(s.config.Chain.Params.ChainID),
		s.host.ID(),
	)

	// Register the identity service protocol
	s.registerIdentityService(identityService)

	// Register the network notify bundle handlers
	s.host.Network().Notify(identityService.GetNotifyBundle())

	return nil
}

// registerFrostService registers the identity service
func (s *Server) registerFrostService(identityService *frost.FrostService) {
	//s.RegisterProtocol(common.IdentityProto, grpcStream)
}

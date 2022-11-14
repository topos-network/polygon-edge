package frost

import (
	"context"
	"fmt"
	"sync"

	"github.com/0xPolygon/go-ibft/core"
	proto "github.com/0xPolygon/polygon-edge/consensus/ibft/frost/messages"
)

type frostStateType uint8

const (
	newRound frostStateType = iota
	keyGeneration
	signing
	finished
)

func (s frostStateType) String() (str string) {
	switch s {
	case newRound:
		str = "new round"
	case keyGeneration:
		str = "key generation"
	case signing:
		str = "signing"
	case finished:
		str = "finished"
	}

	return
}

type state struct {
	roundStarted bool
	generatedKey string
	name         frostStateType
}

func (s *state) clear() {
	s.roundStarted = false
	s.name = newRound
}

type Frost struct {
	// log is the logger instance
	log core.Logger

	// transport is the reference to the
	// Transport implementation
	transport Transport

	state state
}

// NewForkManager is a constructor of ForkManager
func NewFrost(logger core.Logger, transport Transport) *Frost {

	fm := &Frost{
		log:       logger,
		transport: transport,
		state:     state{},
	}

	return fm
}

// sendPreprepareMessage sends out the preprepare message
func (i *Frost) sendNewRoundMessage(message *proto.Message) {
	i.transport.MulticastFrost(message)
}

// DKG Sequence runs the distributed key generation sequence
func (i *Frost) DKGSequence(ctx context.Context) {
	// Set the starting state data
	i.state.clear()

	i.log.Info("Frost DKG sequence started")
	defer i.log.Info("Frost DKG sequence done")

	// for {
	// 	view := i.state.getView()

	// 	i.log.Info("round started", "round", view.Round)

	// 	currentRound := view.Round
	// 	ctxRound, cancelRound := context.WithCancel(ctx)

	// 	i.wg.Add(4)

	// 	// Start the round timer worker
	// 	go i.startRoundTimer(ctxRound, currentRound)

	// 	//	Jump round on proposals from higher rounds
	// 	go i.watchF
	// 	orFutureProposal(ctxRound)

	// 	//	Jump round on certificates
	// 	go i.watchForRoundChangeCertificates(ctxRound)

	// 	// Start the state machine worker
	// 	go i.startRound(ctxRound)

	// 	teardown := func() {
	// 		cancelRound()
	// 		i.wg.Wait()
	// 	}

	// 	select {
	// 	case ev := <-i.newProposal:
	// 		teardown()
	// 		i.log.Info("received future proposal", "round", ev.round)

	// 		i.moveToNewRound(ev.round)
	// 		i.acceptProposal(ev.proposalMessage)
	// 		i.state.setRoundStarted(true)
	// 	case round := <-i.roundCertificate:
	// 		teardown()
	// 		i.log.Info("received future RCC", "round", round)

	// 		i.moveToNewRound(round)
	// 	case <-i.roundExpired:
	// 		teardown()
	// 		i.log.Info("round timeout expired", "round", currentRound)

	// 		newRound := currentRound + 1
	// 		i.moveToNewRound(newRound)

	// 		i.sendRoundChangeMessage(h, newRound)
	// 	case <-i.roundDone:
	// 		// The consensus cycle for the block height is finished.
	// 		// Stop all running worker threads
	// 		teardown()

	// 		return
	// 	case <-ctx.Done():
	// 		teardown()
	// 		i.log.Debug("sequence cancelled")

	// 		return
	// 	}
	// }
}

// SigningSequence runs the Frost sequence for the specified height
func (i *Frost) SigningSequence(ctx context.Context, h uint64) {
	// Set the starting state data
	i.state.clear()
	i.log.Info("Frost signing sequence started", "height", h)
	defer i.log.Info("Frost signing sequence done", "height", h)
}

// IBFTConsensus is a convenience wrapper for the go-ibft package
type FrostConsensus struct {
	*Frost
	wg             sync.WaitGroup
	cancelSequence context.CancelFunc
}

func NewFrostConsensus(
	logger core.Logger,
	transport Transport,
) *FrostConsensus {
	return &FrostConsensus{
		Frost: NewFrost(logger, transport),
		wg:    sync.WaitGroup{},
	}
}

// RunDKGSequence performs sequence for distributed key generation.
func (c *FrostConsensus) RunDKGSequence() <-chan struct{} {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	fmt.Println(">>>>> FROST running DKG sequence:")

	c.cancelSequence = cancel

	c.wg.Add(1)

	go func() {
		defer func() {
			cancel()
			c.wg.Done()
			close(done)
		}()

		c.DKGSequence(ctx)
	}()

	fmt.Println(">>>>> FROST finished with DKG sequence")

	return done
}

// RunFrostSigningSequence starts the underlying frost signing mechanism for the state at the given height.
// It may be called by a single thread at any given time
func (c *FrostConsensus) RunFrostSigningSequence(height uint64) <-chan struct{} {
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	fmt.Println(">>>>> FROST running signing sequence height:", height)

	c.cancelSequence = cancel

	c.wg.Add(1)

	go func() {
		defer func() {
			cancel()
			c.wg.Done()
			close(done)
		}()

		c.SigningSequence(ctx, height)
	}()

	fmt.Println(">>>>> FROST finished with signing sequence height: ", height)

	return done
}

// stopSequence terminates the running Frost sequence gracefully and waits for it to return
func (c *FrostConsensus) stopSequence() {
	c.cancelSequence()
	c.wg.Wait()
}

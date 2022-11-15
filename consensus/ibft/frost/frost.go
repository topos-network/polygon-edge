package frost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xPolygon/go-ibft/core"
	proto "github.com/0xPolygon/polygon-edge/consensus/ibft/frost/messages"
	"github.com/0xPolygon/polygon-edge/validators"
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
	receivedDkgMessages     map[string]string
	receivedSigningMessages map[string]string
	generatedKey            string
	generatedSignature      string
	name                    frostStateType
}

func (s *state) clear() {
	s.name = newRound
	//s.receivedDkgMessages = make(map[string]string)
	s.generatedKey = ""
}

type Frost struct {
	myId []byte

	// log is the logger instance
	log core.Logger

	// transport is the reference to the
	// Transport implementation
	transport Transport

	state state
}

// NewForkManager is a constructor of ForkManager
func NewFrost(id []byte, logger core.Logger, transport Transport) *Frost {
	fm := &Frost{
		myId:      id,
		log:       logger,
		transport: transport,
		state:     state{},
	}

	return fm
}

func (f *Frost) AddFrostMessage(message *proto.FrostMessage) {
	fmt.Println(">>>>>>>>>>>>>> Message received on node: " + string(f.myId) + ",message: " + message.String())
	// Make sure the message is present
	if message == nil {
		return
	}

	if f.state.receivedDkgMessages == nil {
		f.state.receivedDkgMessages = make(map[string]string)
	}
	f.state.receivedDkgMessages[string(message.From)] = string(message.Data)
}

// sendPreprepareMessage sends out the preprepare message
func (i *Frost) sendMessage(message *proto.FrostMessage) {
	i.transport.MulticastFrost(message)
}

// DKG Sequence runs the distributed key generation sequence
func (f *Frost) DKGSequence(ctx context.Context, validators validators.Validators) {
	// Set the starting state data
	f.state.clear()

	f.log.Info("Frost DKG sequence started")

	go func() {
		for {
			// wait for messages from other validators some
			fmt.Println(">>>>>>>>>>>>>> Validators len:", validators.Len(), "messages len: ", len(f.state.receivedDkgMessages))
			if len(f.state.receivedDkgMessages) == validators.Len() {
				for _, v := range f.state.receivedDkgMessages {
					f.state.generatedKey = f.state.generatedKey + v
					fmt.Println(">>>>>>>>>>>>>> Generating key on node:", f.myId, " v=:", v, " key is:", f.state.generatedKey)
				}
				f.log.Info("Frost key has just been generated on node:", f.myId, " key:", f.state.generatedKey)
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	time.Sleep(1000 * time.Millisecond)
	msg := &proto.FrostMessage{
		From: f.myId,
		Type: proto.FrostMessageType_KEY_GENERATION,
		Data: f.myId[0:2],
	}

	f.sendMessage(msg)

	defer f.log.Info("Frost DKG sequence done")
}

// SigningSequence runs the Frost sequence for the specified height
func (i *Frost) SigningSequence(ctx context.Context, h uint64) {
	// Set the starting state data
	defer i.log.Info("Frost signing sequence done", "height", h)
	i.state.clear()
	// todo
	i.log.Info("Frost signing sequence started", "height", h)
}

// IBFTConsensus is a convenience wrapper for the go-ibft package
type FrostConsensus struct {
	*Frost
	wg             sync.WaitGroup
	cancelSequence context.CancelFunc
}

func NewFrostConsensus(
	id []byte,
	logger core.Logger,
	transport Transport,
) *FrostConsensus {
	return &FrostConsensus{
		Frost: NewFrost(id, logger, transport),
		wg:    sync.WaitGroup{},
	}
}

// RunDKGSequence performs sequence for distributed key generation.
func (c *FrostConsensus) RunDKGSequence(validators validators.Validators) <-chan struct{} {
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

		c.DKGSequence(ctx, validators)
	}()

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
func (c *FrostConsensus) ProcessMessage(msg *proto.FrostMessage) {
	c.Frost.AddFrostMessage(msg)
}

// stopSequence terminates the running Frost sequence gracefully and waits for it to return
func (c *FrostConsensus) stopSequence() {
	c.cancelSequence()
	c.wg.Wait()
}

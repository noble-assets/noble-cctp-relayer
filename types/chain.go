package types

import (
	"context"
	"time"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
)

// Chain is an interface for common CCTP source and destination chain operations.
type Chain interface {
	// Name returns the name of the chain.
	Name() string

	// Domain returns the domain ID of the chain.
	Domain() Domain

	// LatestBlockain returns the last queired height of the chain
	LatestBlock() uint64

	// SetLatestBlock sets the latest block
	SetLatestBlock(block uint64)

	// LastFlushedBlock returns the last block included in a flush. In the rare situation of a crash,
	// this block is a good block to start at to catch up on any missed transactions.
	LastFlushedBlock() uint64

	// IsDestinationCaller returns true if the specified destination caller is the minter for the specified domain OR
	// if destination caller is a zero byte array(left empty in deposit for burn message). It also returns a human readable
	// version of the destination caller address provided in the message.
	IsDestinationCaller(destinationCaller []byte) (isCaller bool, readableAddress string)

	// InitializeClients initializes the rpc and or websocket clients.
	InitializeClients(
		ctx context.Context,
		logger log.Logger,
	) error

	// CloseClients is a cleanup function to close any open clients
	CloseClients() error

	// InitializeBroadcaster initializes the minter account info for the chain.
	InitializeBroadcaster(
		ctx context.Context,
		logger log.Logger,
		sequenceMap *SequenceMap,
	) error

	// StartListener starts a listener for observing new CCTP burn messages.
	StartListener(
		ctx context.Context,
		logger log.Logger,
		processingQueue chan *TxState,
		flushOnlyMode bool,
		flushInterval time.Duration,
	)

	// Broadcast broadcasts CCTP mint messages to the chain.
	Broadcast(
		ctx context.Context,
		logger log.Logger,
		msgs []*MessageState,
		sequenceMap *SequenceMap,
		metrics *relayer.PromMetrics,
	) error

	TrackLatestBlockHeight(
		ctx context.Context,
		logger log.Logger,
		metrics *relayer.PromMetrics,
	)

	WalletBalanceMetric(
		ctx context.Context,
		logger log.Logger,
		metrics *relayer.PromMetrics,
	)
}

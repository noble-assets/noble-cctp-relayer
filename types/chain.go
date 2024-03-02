package types

import (
	"context"

	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
)

// Chain is an interface for common CCTP source and destination chain operations.
type Chain interface {
	// Name returns the name of the chain.
	Name() string

	// Domain returns the domain ID of the chain.
	Domain() Domain

	// IsDestinationCaller returns true if the specified destination caller is the minter for the specified domain OR
	// if destination caller is a zero byte array(left empty in deposit for burn message)
	IsDestinationCaller(destinationCaller []byte) bool

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
	)

	// Broadcast broadcasts CCTP mint messages to the chain.
	Broadcast(
		ctx context.Context,
		logger log.Logger,
		msgs []*MessageState,
		sequenceMap *SequenceMap,
	) error

	WalletBalanceMetric(
		ctx context.Context,
		logger log.Logger,
		metric *relayer.PromMetrics,
	)
}

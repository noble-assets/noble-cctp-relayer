package solana

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.Chain = (*Solana)(nil)

type Solana struct {
	endpoints struct {
		RPC string
		WS  string
	}
	messageTransmitter solana.PublicKey

	mu               sync.Mutex
	latestBlock      uint64
	lastFlushedBlock uint64
}

func NewSolana(
	rpcEndpoint string,
	wsEndpoint string,
	messageTransmitter string,
) *Solana {
	return &Solana{
		endpoints: struct {
			RPC string
			WS  string
		}{RPC: rpcEndpoint, WS: wsEndpoint},
		messageTransmitter: solana.MustPublicKeyFromBase58(messageTransmitter),
	}
}

// Name implements the types.Chain interface.
func (s *Solana) Name() string { return "Solana" }

// Domain returns the specific domain for Solana.
// https://developers.circle.com/stablecoins/supported-domains
func (s *Solana) Domain() types.Domain { return 5 }

// LatestBlock implements the types.Chain interface.
func (s *Solana) LatestBlock() uint64 {
	s.mu.Lock()
	block := s.latestBlock
	s.mu.Unlock()
	return block
}

// SetLatestBlock implements the types.Chain interface.
func (s *Solana) SetLatestBlock(block uint64) {
	s.mu.Lock()
	s.latestBlock = block
	s.mu.Unlock()
}

// LastFlushedBlock implements the types.Chain interface.
func (s *Solana) LastFlushedBlock() uint64 { return s.lastFlushedBlock }

// IsDestinationCaller implements the types.Chain interface.
// TODO: Implement!!!
func (s *Solana) IsDestinationCaller(_ []byte) (isCaller bool, readableAddress string) {
	return false, ""
}

// InitializeClients implements the types.Chain interface.
// NOTE: This is left empty intentionally, as there are no Solana clients to initialize.
func (s *Solana) InitializeClients(_ context.Context, _ log.Logger) error {
	return nil
}

// CloseClients implements the types.Chain interface.
// NOTE: This is left empty intentionally, as there are no Solana clients to close.
func (s *Solana) CloseClients() error { return nil }

// InitializeBroadcaster implements the types.Chain interface.
// TODO: Implement!!!
func (s *Solana) InitializeBroadcaster(_ context.Context, _ log.Logger, _ *types.SequenceMap) error {
	return nil
}

// StartListener connects to a Solana WebSocket and listens for finalized
// transactions on the CCTP Message Transmitter program. It then parses each
// received transaction's instructions for sent CCTP messages.
// TODO: Handle flushing missed transactions on startup.
func (s *Solana) StartListener(ctx context.Context, logger log.Logger, processingQueue chan *types.TxState, _ bool, _ time.Duration) {
	client, err := ws.Connect(ctx, s.endpoints.WS)
	if err != nil {
		panic(err)
	}

	subscription, err := client.LogsSubscribeMentions(s.messageTransmitter, rpc.CommitmentFinalized)
	if err != nil {
		panic(err)
	}

	for {
		log, err := subscription.Recv()
		if err != nil {
			panic(err)
		}

		txHash := log.Value.Signature.String()
		event, err := s.ParseTransaction(ctx, txHash)
		if err != nil {
			panic(err)
		}

		if event != nil {
			logger.Info(fmt.Sprintf("Found a new transfer to Domain %d", event.DestDomain), "tx", txHash)

			processingQueue <- &types.TxState{
				TxHash: txHash,
				Msgs:   []*types.MessageState{event},
			}
		}
	}
}

// Broadcast implements the types.Chain interfaces.
// TODO: Implement!!!
func (s *Solana) Broadcast(_ context.Context, _ log.Logger, _ []*types.MessageState, _ *types.SequenceMap, _ *relayer.PromMetrics) error {
	return nil
}

// TrackLatestBlockHeight continuously queries Solana for the latest block height.
// TODO: Ensure we are querying finalized blocks only!
func (s *Solana) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, metrics *relayer.PromMetrics) {
	domain := fmt.Sprint(s.Domain())

	updateBlockHeight := func() {
		blockHeight, err := s.GetBlockHeight(ctx)
		if err != nil {
			logger.Error("Unable to query Solana's block height", "err", err)
		} else {
			s.SetLatestBlock(blockHeight)
			if metrics != nil {
				metrics.SetLatestHeight(s.Name(), domain, int64(blockHeight))
			}
		}
	}

	updateBlockHeight()

	for {
		// TODO: Is 1s fast enough?
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-timer.C:
			updateBlockHeight()
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// WalletBalanceMetric implements the types.Chain interface.
// TODO: Implement!!!
func (s *Solana) WalletBalanceMetric(_ context.Context, _ log.Logger, _ *relayer.PromMetrics) {}

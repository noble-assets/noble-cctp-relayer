package solana

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/solana/generated/message_transmitter"
	"github.com/strangelove-ventures/noble-cctp-relayer/solana/generated/token_messenger_minter"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.Chain = (*Solana)(nil)

type Solana struct {
	endpoints struct {
		RPC string
		WS  string
	}
	rpcClient *rpc.Client
	wallet    solana.Wallet

	messageTransmitter   solana.PublicKey
	tokenMessengerMinter solana.PublicKey
	fiatToken            solana.PublicKey
	remoteTokens         map[types.Domain]solana.PublicKey

	mu               sync.Mutex
	latestBlock      uint64
	lastFlushedBlock uint64
}

func NewSolana(cfg Config) *Solana {
	wallet, err := solana.WalletFromPrivateKeyBase58(cfg.PrivateKey)
	if err != nil {
		panic(err)
	}

	messageTransmitter := solana.MustPublicKeyFromBase58(cfg.MessageTransmitter)
	message_transmitter.SetProgramID(messageTransmitter)

	tokenMessengerMinter := solana.MustPublicKeyFromBase58(cfg.TokenMessengerMinter)
	token_messenger_minter.SetProgramID(tokenMessengerMinter)

	remoteTokens := make(map[types.Domain]solana.PublicKey)
	for domain, rawRemoteToken := range cfg.RemoteTokens {
		if strings.HasPrefix(rawRemoteToken, "0x") {
			remoteToken := make([]byte, 32)
			tmpRemoteToken := common.FromHex(rawRemoteToken)
			copy(remoteToken[32-len(tmpRemoteToken):], tmpRemoteToken)

			remoteTokens[domain] = solana.PublicKeyFromBytes(remoteToken)
		} else {
			panic("unsupported remote token: " + rawRemoteToken)
		}
	}

	return &Solana{
		endpoints: struct {
			RPC string
			WS  string
		}{RPC: cfg.RPC, WS: cfg.WS},
		wallet:               *wallet,
		messageTransmitter:   messageTransmitter,
		tokenMessengerMinter: tokenMessengerMinter,
		fiatToken:            solana.MustPublicKeyFromBase58(cfg.FiatToken),
		remoteTokens:         remoteTokens,
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

// IsDestinationCaller checks if the relayer wallet is the specified destination caller.
func (s *Solana) IsDestinationCaller(destinationCaller []byte) (isCaller bool, readableAddress string) {
	return bytes.Equal(destinationCaller, s.wallet.PublicKey().Bytes()), s.wallet.PublicKey().String()
}

// InitializeClients creates a new client for a Solana RPC endpoint.
func (s *Solana) InitializeClients(_ context.Context, _ log.Logger) error {
	s.rpcClient = rpc.New(s.endpoints.RPC)
	return nil
}

// CloseClients implements the types.Chain interface.
// NOTE: This is left empty intentionally, as there are no Solana clients to close.
func (s *Solana) CloseClients() error { return nil }

// InitializeBroadcaster implements the types.Chain interface.
// NOTE: This is left empty intentionally, as there is no Solana broadcaster to initialize.
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

// Broadcast receives and executes a list of messages from other domains to Solana.
func (s *Solana) Broadcast(ctx context.Context, _ log.Logger, inputs []*types.MessageState, _ *types.SequenceMap, _ *relayer.PromMetrics) error {
	var instructions []solana.Instruction

	for _, input := range inputs {
		instruction := message_transmitter.NewReceiveMessageInstructionBuilder()

		instruction.SetParams(message_transmitter.ReceiveMessageParams{
			Message:     input.MsgSentBytes,
			Attestation: common.FromHex(input.Attestation),
		})
		err := instruction.SetAccounts(s.GetReceiveMessageAccounts(input))
		if err != nil {
			return err
		}

		instructions = append(instructions, instruction.Build())
	}

	recent, err := s.rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return err
	}
	tx, err := solana.NewTransaction(instructions, recent.Value.Blockhash)
	if err != nil {
		return err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key == s.wallet.PublicKey() {
			return &s.wallet.PrivateKey
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = s.rpcClient.SendTransaction(ctx, tx)
	return err
}

// TrackLatestBlockHeight continuously queries Solana for the latest block height.
func (s *Solana) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, metrics *relayer.PromMetrics) {
	domain := fmt.Sprint(s.Domain())

	updateBlockHeight := func() {
		blockHeight, err := s.rpcClient.GetBlockHeight(ctx, rpc.CommitmentFinalized)
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

// WalletBalanceMetric continuously queries Solana for the SOL balance of the relayer wallet.
func (s *Solana) WalletBalanceMetric(ctx context.Context, logger log.Logger, metrics *relayer.PromMetrics) {
	updateBalance := func() {
		res, err := s.rpcClient.GetBalance(ctx, s.wallet.PublicKey(), rpc.CommitmentFinalized)
		if err != nil {
			logger.Error("Unable to query relayer wallet balance", "err", err)
		} else if metrics != nil {
			balance := float64(res.Value) / 1e9
			metrics.SetWalletBalance(s.Name(), s.wallet.PublicKey().String(), "SOL", balance)
		}
	}

	updateBalance()

	for {
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
			updateBalance()
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

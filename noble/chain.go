package noble

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.Chain = (*Noble)(nil)

type Noble struct {
	// from config
	chainID               string
	rpcURL                string
	privateKey            *secp256k1.PrivKey
	minterAddress         string
	accountNumber         uint64
	startBlock            uint64
	lookbackPeriod        uint64
	workers               uint32
	gasLimit              uint64
	txMemo                string
	maxRetries            int
	retryIntervalSeconds  int
	blockQueueChannelSize uint64
	minAmount             uint64

	mu sync.Mutex

	cc *cosmos.CosmosProvider

	latestBlock      uint64
	lastFlushedBlock uint64
}

func NewChain(
	rpcURL string,
	chainID string,
	privateKey string,
	startBlock uint64,
	lookbackPeriod uint64,
	workers uint32,
	gasLimit uint64,
	txMemo string,
	maxRetries int,
	retryIntervalSeconds int,
	blockQueueChannelSize uint64,
	minAmount uint64,
) (*Noble, error) {
	keyBz, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse noble private key: %w", err)
	}

	privKey := secp256k1.PrivKey{Key: keyBz}

	address := privKey.PubKey().Address()
	minterAddress := sdk.MustBech32ifyAddressBytes("noble", address)

	return &Noble{
		chainID:               chainID,
		rpcURL:                rpcURL,
		startBlock:            startBlock,
		lookbackPeriod:        lookbackPeriod,
		workers:               workers,
		privateKey:            &privKey,
		minterAddress:         minterAddress,
		gasLimit:              gasLimit,
		txMemo:                txMemo,
		maxRetries:            maxRetries,
		retryIntervalSeconds:  retryIntervalSeconds,
		blockQueueChannelSize: blockQueueChannelSize,
		minAmount:             minAmount,
	}, nil
}

func (n *Noble) AccountInfo(ctx context.Context) (uint64, uint64, error) {
	res, err := authtypes.NewQueryClient(n.cc).Account(ctx, &authtypes.QueryAccountRequest{
		Address: n.minterAddress,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query account for noble: %w", err)
	}
	var acc authtypes.AccountI
	if err := n.cc.Cdc.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, 0, fmt.Errorf("unable to unpack account for noble: %w", err)
	}

	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

func (n *Noble) Name() string {
	return "Noble"
}

func (n *Noble) Domain() types.Domain {
	return 4
}

func (n *Noble) LatestBlock() uint64 {
	n.mu.Lock()
	block := n.latestBlock
	n.mu.Unlock()
	return block
}

func (n *Noble) SetLatestBlock(block uint64) {
	n.mu.Lock()
	n.latestBlock = block
	n.mu.Unlock()
}

func (n *Noble) LastFlushedBlock() uint64 {
	return n.lastFlushedBlock
}

func (n *Noble) IsDestinationCaller(destinationCaller []byte) (isCaller bool, readableAddress string) {
	zeroByteArr := make([]byte, 32)

	if bytes.Equal(destinationCaller, zeroByteArr) {
		return true, ""
	}

	bech32DestinationCaller, err := decodeDestinationCaller(destinationCaller)
	if err != nil {
		return false, bech32DestinationCaller
	}

	return bech32DestinationCaller == n.minterAddress, bech32DestinationCaller
}

// DecodeDestinationCaller transforms an encoded Noble cctp address into a noble bech32 address
// left padded input -> bech32 output
func decodeDestinationCaller(input []byte) (string, error) {
	if len(input) <= 12 {
		return "", errors.New("destinationCaller is too short")
	}
	output, err := bech32.ConvertAndEncode("noble", input[12:])
	if err != nil {
		return "", errors.New("unable to encode destination caller")
	}
	return output, nil
}

func (n *Noble) InitializeClients(ctx context.Context, logger log.Logger) error {
	var err error
	n.cc, err = cosmos.NewProvider(n.rpcURL)
	if err != nil {
		return fmt.Errorf("unable to build cosmos provider for noble: %w", err)
	}
	return nil
}

func (n *Noble) CloseClients() error {
	if n.cc != nil && n.cc.RPCClient.IsRunning() {
		err := n.cc.RPCClient.Stop()
		if err != nil {
			return fmt.Errorf("error stopping noble rpc client: %w", err)
		}
	}
	return nil
}

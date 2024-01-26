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
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.Chain = (*Noble)(nil)

type Noble struct {
	cc      *cosmos.CosmosProvider
	chainID string

	privateKey    *secp256k1.PrivKey
	minterAddress string
	accountNumber uint64

	startBlock     uint64
	lookbackPeriod uint64
	workers        uint32

	gasLimit             uint64
	txMemo               string
	maxRetries           int
	retryIntervalSeconds int

	mu sync.Mutex
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
) (*Noble, error) {
	cc, err := cosmos.NewProvider(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("unable to build cosmos provider for noble: %w", err)
	}

	keyBz, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse noble private key: %w", err)
	}

	privKey := secp256k1.PrivKey{Key: keyBz}

	address := privKey.PubKey().Address()
	minterAddress := sdk.MustBech32ifyAddressBytes("noble", address)

	return &Noble{
		cc:                   cc,
		chainID:              chainID,
		startBlock:           startBlock,
		lookbackPeriod:       lookbackPeriod,
		workers:              workers,
		privateKey:           &privKey,
		minterAddress:        minterAddress,
		gasLimit:             gasLimit,
		txMemo:               txMemo,
		maxRetries:           maxRetries,
		retryIntervalSeconds: retryIntervalSeconds,
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

func (n *Noble) IsDestinationCaller(destinationCaller []byte) bool {
	zeroByteArr := make([]byte, 32)

	if bytes.Equal(destinationCaller, zeroByteArr) {
		return true
	}

	bech32DestinationCaller, err := decodeDestinationCaller(destinationCaller)
	if err != nil {
		return false
	}

	return bech32DestinationCaller == n.minterAddress
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

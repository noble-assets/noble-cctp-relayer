package ethereum

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"embed"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"cosmossdk.io/log"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

//go:embed abi/MessageTransmitter.json
var content embed.FS

var _ types.Chain = (*Ethereum)(nil)

type Ethereum struct {
	// from config
	name                      string
	chainID                   int64
	domain                    types.Domain
	rpcURL                    string
	wsURL                     string
	messageTransmitterAddress string
	startBlock                uint64
	lookbackPeriod            uint64
	privateKey                *ecdsa.PrivateKey
	minterAddress             string
	maxRetries                int
	retryIntervalSeconds      int
	minAmount                 uint64
	MetricsDenom              string
	MetricsExponent           int

	mu sync.Mutex

	wsClient  *ethclient.Client
	rpcClient *ethclient.Client

	latestBlock      uint64
	lastFlushedBlock uint64
}

func NewChain(
	name string,
	domain types.Domain,
	chainID int64,
	rpcURL string,
	wsURL string,
	messageTransmitterAddress string,
	startBlock uint64,
	lookbackPeriod uint64,
	privateKey string,
	maxRetries int,
	retryIntervalSeconds int,
	minAmount uint64,
	metricsDenom string,
	metricsExponent int,
) (*Ethereum, error) {
	privEcdsaKey, ethereumAddress, err := GetEcdsaKeyAddress(privateKey)
	if err != nil {
		return nil, err
	}
	return &Ethereum{
		name:                      name,
		chainID:                   chainID,
		domain:                    domain,
		rpcURL:                    rpcURL,
		wsURL:                     wsURL,
		messageTransmitterAddress: messageTransmitterAddress,
		startBlock:                startBlock,
		lookbackPeriod:            lookbackPeriod,
		privateKey:                privEcdsaKey,
		minterAddress:             ethereumAddress,
		maxRetries:                maxRetries,
		retryIntervalSeconds:      retryIntervalSeconds,
		minAmount:                 minAmount,
		MetricsDenom:              metricsDenom,
		MetricsExponent:           metricsExponent,
	}, nil
}

func (e *Ethereum) Name() string {
	return e.name
}

func (e *Ethereum) Domain() types.Domain {
	return e.domain
}

func (e *Ethereum) LatestBlock() uint64 {
	e.mu.Lock()
	block := e.latestBlock
	e.mu.Unlock()
	return block
}

func (e *Ethereum) SetLatestBlock(block uint64) {
	e.mu.Lock()
	e.latestBlock = block
	e.mu.Unlock()
}

func (e *Ethereum) LastFlushedBlock() uint64 {
	return e.lastFlushedBlock
}

func (e *Ethereum) IsDestinationCaller(destinationCaller []byte) (isCaller bool, readableAddress string) {
	zeroByteArr := make([]byte, 32)

	decodedMinter, err := hex.DecodeString(strings.ReplaceAll(e.minterAddress, "0x", ""))
	if err != nil && bytes.Equal(destinationCaller, zeroByteArr) {
		return true, ""
	}

	decodedMinterPadded := make([]byte, 32)
	copy(decodedMinterPadded[12:], decodedMinter)

	encodedCaller := "0x" + hex.EncodeToString(destinationCaller)[24:]

	if bytes.Equal(destinationCaller, zeroByteArr) || bytes.Equal(destinationCaller, decodedMinterPadded) {
		return true, encodedCaller
	}
	return false, encodedCaller
}
func (e *Ethereum) InitializeClients(ctx context.Context, logger log.Logger) error {
	var err error

	e.wsClient, err = ethclient.DialContext(ctx, e.wsURL)
	if err != nil {
		return fmt.Errorf("unable to initialize websocket ethereum client; err: %w", err)
	}

	e.rpcClient, err = ethclient.DialContext(ctx, e.rpcURL)
	if err != nil {
		return fmt.Errorf("unable to initialize rpc ethereum client; err: %w", err)
	}
	return nil
}

func (e *Ethereum) CloseClients() error {
	if e.wsClient != nil {
		e.wsClient.Close()
	}
	if e.rpcClient != nil {
		e.rpcClient.Close()
	}
	return nil
}

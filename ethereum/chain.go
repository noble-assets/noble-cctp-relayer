package ethereum

import (
	"bytes"
	"crypto/ecdsa"
	"embed"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

//go:embed abi/MessageTransmitter.json
var content embed.FS

var _ types.Chain = (*Ethereum)(nil)

type Ethereum struct {
	name                      string
	chainID                   int64
	domain                    types.Domain
	rpcURL                    string
	wsURL                     string
	messageTransmitterAddress string
	startBlock                uint64
	endBlock                  uint64
	lookbackPeriod            uint64
	privateKey                *ecdsa.PrivateKey
	minterAddress             string
	maxRetries                int
	retryIntervalSeconds      int
	minAmount                 uint64

	mu sync.Mutex
}

func NewChain(
	name string,
	domain types.Domain,
	chainID int64,
	rpcURL string,
	wsURL string,
	messageTransmitterAddress string,
	startBlock uint64,
	endBlock uint64,
	lookbackPeriod uint64,
	privateKey string,
	maxRetries int,
	retryIntervalSeconds int,
	minAmount uint64,
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
		endBlock:                  endBlock,
		lookbackPeriod:            lookbackPeriod,
		privateKey:                privEcdsaKey,
		minterAddress:             ethereumAddress,
		maxRetries:                maxRetries,
		retryIntervalSeconds:      retryIntervalSeconds,
		minAmount:                 minAmount,
	}, nil
}

func (e *Ethereum) Name() string {
	return e.name
}

func (e *Ethereum) Domain() types.Domain {
	return e.domain
}

func (e *Ethereum) IsDestinationCaller(destinationCaller []byte) bool {
	zeroByteArr := make([]byte, 32)

	decodedMinter, err := hex.DecodeString(strings.ReplaceAll(e.minterAddress, "0x", ""))
	if err != nil && bytes.Equal(destinationCaller, zeroByteArr) {
		return true
	}

	decodedMinterPadded := make([]byte, 32)
	copy(decodedMinterPadded[12:], decodedMinter)

	return bytes.Equal(destinationCaller, zeroByteArr) || bytes.Equal(destinationCaller, decodedMinterPadded)
}

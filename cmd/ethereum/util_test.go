package ethereum_test

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func init() {
	cfg = config.Parse("../../.ignore/unit_tests.yaml")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

func TestGetEthereumAccountNonce(t *testing.T) {
	_, err := ethereum.GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, "0x4996f29b254c77972fff8f25e6f7797b3c9a0eb6")
	require.Nil(t, err)
}

// Return public ecdsa key and address given the private key
func TestGetEcdsaKeyAddress(t *testing.T) {
	key, addr, err := ethereum.GetEcdsaKeyAddress(cfg.Networks.Minters[0].MinterPrivateKey)
	require.NotNil(t, key)
	require.NotNil(t, addr)
	require.Nil(t, err)
}

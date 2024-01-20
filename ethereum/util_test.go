package ethereum_test

import (
	"os"
	"testing"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

func init() {
	var err error
	cfg, err = cmd.Parse("../.ignore/unit_tests.yaml")
	if err != nil {
		panic(err)
	}

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.TxState, 10000)
}

func TestGetEthereumAccountNonce(t *testing.T) {
	_, err := ethereum.GetEthereumAccountNonce(cfg.Chains["ethereum"].(*ethereum.ChainConfig).RPC, "0x4996f29b254c77972fff8f25e6f7797b3c9a0eb6")
	require.Nil(t, err)
}

// Return public ecdsa key and address given the private key
func TestGetEcdsaKeyAddress(t *testing.T) {
	key, addr, err := ethereum.GetEcdsaKeyAddress(cfg.Chains["ethereum"].(*ethereum.ChainConfig).MinterPrivateKey)
	require.NotNil(t, key)
	require.NotNil(t, addr)
	require.Nil(t, err)
}

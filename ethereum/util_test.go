package ethereum_test

import (
	"testing"

	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
	"github.com/stretchr/testify/require"
)

func TestGetEthereumAccountNonce(t *testing.T) {
	a, _ := testutil.ConfigSetup(t)
	ethConfig := a.Config.Chains["ethereum"].(*ethereum.ChainConfig)

	_, err := ethereum.GetEthereumAccountNonce(ethConfig.RPC, "0x4996f29b254c77972fff8f25e6f7797b3c9a0eb6")
	require.Nil(t, err)
}

// Return public ecdsa key and address given the private key
func TestGetEcdsaKeyAddress(t *testing.T) {
	a, _ := testutil.ConfigSetup(t)
	ethConfig := a.Config.Chains["ethereum"].(*ethereum.ChainConfig)

	key, addr, err := ethereum.GetEcdsaKeyAddress(ethConfig.MinterPrivateKey)
	require.NotNil(t, key)
	require.NotNil(t, addr)
	require.Nil(t, err)
}

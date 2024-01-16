package ethereum_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/stretchr/testify/require"
)

func TestEthUsedNonce(t *testing.T) {
	sourceDomain := uint32(4)
	nonce := uint64(2970)

	key := append(
		common.LeftPadBytes((big.NewInt(int64(sourceDomain))).Bytes(), 4),
		common.LeftPadBytes((big.NewInt(int64(nonce))).Bytes(), 8)...,
	)

	require.Equal(t, []byte("\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x0B\x9A"), key)

	client, err := ethclient.Dial("https://mainnet.infura.io/v3/e44b63a541e94b30a74a97447518c0ec")
	require.NoError(t, err)
	defer client.Close()

	messageTransmitter, err := ethereum.NewMessageTransmitter(common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81"), client)
	require.NoError(t, err)

	co := &bind.CallOpts{
		Pending: true,
		Context: context.TODO(),
	}

	response, err := messageTransmitter.UsedNonces(co, [32]byte(crypto.Keccak256(key)))
	require.NoError(t, err)

	require.Equal(t, big.NewInt(1), response)
}

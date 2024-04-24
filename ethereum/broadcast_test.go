package ethereum_test

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
)

func TestEthUsedNonce(t *testing.T) {
	err := godotenv.Load(testutil.EnvFile)
	require.NoError(t, err)

	sourceDomain := uint32(4)
	nonce := uint64(612)

	key := append(
		common.LeftPadBytes((big.NewInt(int64(sourceDomain))).Bytes(), 4),
		common.LeftPadBytes((big.NewInt(int64(nonce))).Bytes(), 8)...,
	)

	client, err := ethclient.Dial(os.Getenv("SEPOLIA_RPC"))
	require.NoError(t, err)
	defer client.Close()

	messageTransmitter, err := contracts.NewMessageTransmitter(common.HexToAddress("0x7865fAfC2db2093669d92c0F33AeEF291086BEFD"), client)
	require.NoError(t, err)

	co := &bind.CallOpts{
		Pending: true,
		Context: context.TODO(),
	}

	response, err := messageTransmitter.UsedNonces(co, [32]byte(crypto.Keccak256(key)))
	require.NoError(t, err)

	require.Equal(t, big.NewInt(1), response)
}

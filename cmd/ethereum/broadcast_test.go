package ethereum_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEthUsedNonce(t *testing.T) {
	sourceDomain := uint32(4)
	nonce := uint64(5)

	key := append(
		common.LeftPadBytes((big.NewInt(int64(sourceDomain))).Bytes(), 4),
		common.LeftPadBytes((big.NewInt(int64(nonce))).Bytes(), 8)...,
	)

	require.Equal(t, []byte("\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x05"), key)

}

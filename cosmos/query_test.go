package cosmos_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
)

func TestUsedNonce(t *testing.T) {
	cc, err := cosmos.NewProvider("https://rpc.noble.strange.love:443")
	require.NoError(t, err)

	used, err := cc.QueryUsedNonce(context.TODO(), 0, 15365)
	require.NoError(t, err)
	require.True(t, used)

	used, err = cc.QueryUsedNonce(context.TODO(), 0, 100)
	require.NoError(t, err)
	require.False(t, used)
}

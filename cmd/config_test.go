package cmd_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
)

func TestConfig(t *testing.T) {
	file, err := cmd.ParseConfig("../config/sample-config.yaml")
	require.NoError(t, err, "Error parsing config")

	// assert noble chainConfig correctly parsed
	var nobleType any = file.Chains["noble"]
	_, ok := nobleType.(*noble.ChainConfig)
	require.True(t, ok)

	// assert ethereum chainConfig correctly parsed
	var ethType any = file.Chains["ethereum"]
	_, ok = ethType.(*ethereum.ChainConfig)
	require.True(t, ok)
}

func TestBlockQueueChannelSize(t *testing.T) {
	file, err := cmd.ParseConfig("../config/sample-config.yaml")
	require.NoError(t, err, "Error parsing config")

	var nobleCfg any = file.Chains["noble"]
	n, ok := nobleCfg.(*noble.ChainConfig)
	require.True(t, ok)

	// block-queue-channel-size is set to 1000000 in sample-config
	expected := uint64(1000000)

	require.Equal(t, expected, n.BlockQueueChannelSize)
}

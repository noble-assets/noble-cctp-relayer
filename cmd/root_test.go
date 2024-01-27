package cmd_test

import (
	"testing"

	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	file, err := cmd.Parse("../config/sample-config.yaml")
	require.NoError(t, err, "Error parsing config")

	// assert noble chainConfig correctly parsed
	var nobleType interface{} = file.Chains["noble"]
	_, ok := nobleType.(*noble.ChainConfig)
	require.True(t, ok)

	// assert ethereum chainConfig correctly parsed
	var ethType interface{} = file.Chains["ethereum"]
	_, ok = ethType.(*ethereum.ChainConfig)
	require.True(t, ok)
}

package cmd_test

import (
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func init() {
	cmd.Cfg.AttestationBaseUrl = "https://iris-api-sandbox.circle.com/attestations/"
	cmd.Cfg.Networks.Noble.ChainId = "grand-1"
	cmd.Cfg.Networks.Noble.RPC = "https://rpc.testnet.noble.strange.love"
}

func TestAttestationIsReady(t *testing.T) {

	att := types.Attestation{
		Key: "85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe",
	}
	resp := cmd.CheckAttestation(&att)
	require.Equal(t, true, resp)
}

func TestMint(t *testing.T) {

	att := types.Attestation{
		Key: "85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe",
	}
	resp := cmd.CheckAttestation(&att)
	require.Equal(t, true, resp)

	err := cmd.Mint(att)
	require.Nil(t, err)
}

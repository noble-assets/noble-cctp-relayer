package circle_test

import (
	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func init() {
	cmd.Cfg.AttestationBaseUrl = "https://iris-api-sandbox.circle.com/attestations/"
	cmd.Cfg.Networks.Destination.Noble.ChainId = "grand-1"
	cmd.Cfg.Networks.Destination.Noble.RPC = "https://rpc.testnet.noble.strange.love"
	cmd.Logger = log.NewLogger(os.Stdout)
}

func TestAttestationIsReady(t *testing.T) {
	resp, found := cmd.CheckAttestation("0x85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe")
	require.Equal(t, "complete", resp.Status)
	require.True(t, found)
}

func TestAttestationNotFound(t *testing.T) {
	resp, found := cmd.CheckAttestation("not an attestation")
	require.Nil(t, resp)
	require.False(t, found)
}

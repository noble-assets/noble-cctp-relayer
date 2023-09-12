package cmd_test

import (
	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
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

func TestProcessSuccess(t *testing.T) {
	// TODO
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

func TestBroadcastSuccess(t *testing.T) {

	messageState := types.MessageState{
		IrisLookupId: "0x85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe",
		Attestation:  "0xbf6b247d7f0bd4e6e30029d806cff5a6808d970180b6b7f81cf0636132984d2b148d6dc1d1fa78fc3210e46b4f7eac71a1e917814639ea327292869aba676c921b9099bb6bd35f02a7430338c053ad694c8b4d0ce8b19e97872904eabfa5d6f64d1b0e2af4dda84e72bf21390640f48ad9da3580d3da7dc349569665163a2879091c", // hex
		MsgSentBytes: nil,
	}

	txResponse, err := cmd.BroadcastNoble(&messageState)
	require.Nil(t, err)
	require.Equal(t, 0, txResponse.Code)

}

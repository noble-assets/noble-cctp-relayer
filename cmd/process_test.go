package cmd_test

import (
	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
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
	// process tests for each state
}

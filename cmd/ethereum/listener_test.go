package ethereum_test

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var cfg config.Config
var logger log.Logger
var processingQueue chan *types.MessageState

func init() {
	cfg = config.Parse("/Users/joel/src/noble-cctp-relayer/.ignore/testing.yaml")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

func TestStartListener(t *testing.T) {

	go eth.StartListener(cfg, logger, processingQueue)

	msg := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupId: "dad7bd811c877720b137f1a633c9acee528d91146c21fd29d574890c733bc07b",
		Type:         "mint",
		Status:       "created",
		SourceDomain: 0,
		DestDomain:   4,
		SourceTxHash: "0x04882ea24ba2ab9131d54883c0693f117b0330b958edc61c1352741269d4f2a8",
	}
	require.Equal(t, expectedMsg.IrisLookupId, msg.IrisLookupId)
	require.Equal(t, expectedMsg.Type, msg.Type)
	require.Equal(t, expectedMsg.Status, msg.Status)
	require.Equal(t, expectedMsg.SourceDomain, msg.SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, msg.DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, msg.SourceTxHash)

}

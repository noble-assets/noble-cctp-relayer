package noble

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var cfg config.Config
var logger log.Logger
var processingQueue chan *types.MessageState

func init() {
	cfg = config.Parse("../../.ignore/unit_tests.yaml")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

func TestStartListener(t *testing.T) {
	cfg.Networks.Source.Noble.StartBlock = 3273557
	cfg.Networks.Source.Noble.LookbackPeriod = 0
	go StartListener(cfg, logger, processingQueue)

	time.Sleep(5 * time.Second)

	msg := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupId: "efe7cea3fd4785c3beab7f37876bdd48c5d4689c84d85a250813a2a7f01fe765",
		Type:         "mint",
		Status:       "created",
		SourceDomain: 4,
		DestDomain:   0,
		SourceTxHash: "5002A249B1353FA59C1660EBAE5FA7FC652AC1E77F69CEF3A4533B0DF2864012",
	}
	require.Equal(t, expectedMsg.IrisLookupId, msg.IrisLookupId)
	require.Equal(t, expectedMsg.Type, msg.Type)
	require.Equal(t, expectedMsg.Status, msg.Status)
	require.Equal(t, expectedMsg.SourceDomain, msg.SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, msg.DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, msg.SourceTxHash)

}

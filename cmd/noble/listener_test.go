package noble_test

import (
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

var cfg config.Config
var logger log.Logger
var processingQueue chan *types.TxState

func init() {
	cfg = config.Parse("../../.ignore/unit_tests.yaml")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))
	processingQueue = make(chan *types.TxState, 10000)
	cfg.Networks.Source.Noble.Workers = 1
}

func TestStartListener(t *testing.T) {
	cfg.Networks.Source.Noble.StartBlock = 3273557
	go noble.StartListener(cfg, logger, processingQueue)

	time.Sleep(20 * time.Second)

	tx := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupId: "efe7cea3fd4785c3beab7f37876bdd48c5d4689c84d85a250813a2a7f01fe765",
		Status:       "created",
		SourceDomain: 4,
		DestDomain:   0,
		SourceTxHash: "5002A249B1353FA59C1660EBAE5FA7FC652AC1E77F69CEF3A4533B0DF2864012",
	}
	require.Equal(t, expectedMsg.IrisLookupId, tx.Msgs[0].IrisLookupId)
	require.Equal(t, expectedMsg.Status, tx.Msgs[0].Status)
	require.Equal(t, expectedMsg.SourceDomain, tx.Msgs[0].SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, tx.Msgs[0].DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, tx.Msgs[0].SourceTxHash)

}

package ethereum_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

var cfg *types.Config
var logger log.Logger
var processingQueue chan *types.TxState

func init() {
	var err error
	cfg, err = cmd.Parse("../.ignore/unit_tests.yaml")
	if err != nil {
		panic(err)
	}

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.TxState, 10000)
}

// tests for a historical log
func TestStartListener(t *testing.T) {
	ethCfg := ethereum.ChainConfig{
		StartBlock:     9702735,
		LookbackPeriod: 0,
	}
	eth, err := ethCfg.Chain("ethereum")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go eth.StartListener(ctx, logger, processingQueue)

	time.Sleep(5 * time.Second)

	tx := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupId: "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:       "created",
		SourceDomain: 0,
		DestDomain:   4,
		SourceTxHash: "0xe1d7729de300274ee3a2fd20ba179b14a8e3ffcd9d847c506b06760f0dad7802",
	}
	require.Equal(t, expectedMsg.IrisLookupId, tx.Msgs[0].IrisLookupId)
	require.Equal(t, expectedMsg.Status, tx.Msgs[0].Status)
	require.Equal(t, expectedMsg.SourceDomain, tx.Msgs[0].SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, tx.Msgs[0].DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, tx.Msgs[0].SourceTxHash)

}

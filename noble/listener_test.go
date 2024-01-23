package noble_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

var cfg *types.Config
var logger log.Logger
var processingQueue chan *types.TxState

func init() {
	var err error
	cfg, err = cmd.Parse("../.ignore/testnet.yaml")
	if err != nil {
		panic(err)
	}

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))
	processingQueue = make(chan *types.TxState, 10000)
	cfg.Chains["noble"].(*noble.ChainConfig).Workers = 1
}

func TestStartListener(t *testing.T) {
	cfg.Chains["noble"].(*noble.ChainConfig).StartBlock = 3273557
	n, err := cfg.Chains["noble"].(*noble.ChainConfig).Chain("noble")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go n.StartListener(ctx, logger, processingQueue)

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

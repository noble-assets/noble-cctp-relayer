package ethereum_test

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

// TODO: update test. This test is currently outdated as the RPC endpoints likely won't have this much history
func TestStartListener(t *testing.T) {
	a, _ := testutil.ConfigSetup(t)

	ethConfig := a.Config.Chains["ethereum"].(*ethereum.ChainConfig)
	ethConfig.StartBlock = 9702735
	ethConfig.LookbackPeriod = 0

	eth, err := ethConfig.Chain("ethereum")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processingQueue := make(chan *types.TxState, 10000)

	go eth.StartListener(ctx, a.Logger, processingQueue)

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

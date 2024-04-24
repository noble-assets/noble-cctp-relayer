package noble_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// TODO: update test to not rely on live node with block history
func TestStartListener(t *testing.T) {
	a, _ := testutil.ConfigSetup(t)
	nobleCfg := a.Config.Chains["noble"].(*noble.ChainConfig)

	nobleCfg.StartBlock = 3273557

	n, err := nobleCfg.Chain("noble")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processingQueue := make(chan *types.TxState, 10000)

	go n.StartListener(ctx, a.Logger, processingQueue, 0)

	time.Sleep(20 * time.Second)

	tx := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupID: "efe7cea3fd4785c3beab7f37876bdd48c5d4689c84d85a250813a2a7f01fe765",
		Status:       "created",
		SourceDomain: 4,
		DestDomain:   0,
		SourceTxHash: "5002A249B1353FA59C1660EBAE5FA7FC652AC1E77F69CEF3A4533B0DF2864012",
	}
	require.Equal(t, expectedMsg.IrisLookupID, tx.Msgs[0].IrisLookupID)
	require.Equal(t, expectedMsg.Status, tx.Msgs[0].Status)
	require.Equal(t, expectedMsg.SourceDomain, tx.Msgs[0].SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, tx.Msgs[0].DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, tx.Msgs[0].SourceTxHash)
}

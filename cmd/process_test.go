package cmd_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

// var a *cmd.AppState
var processingQueue chan *types.TxState

// new log -> create state entry (not a real message, will just create state)
func TestProcessNewLog(t *testing.T) {
	a, registeredDomains := testutil.ConfigSetup(t)

	sequenceMap := types.NewSequenceMap()
	processingQueue = make(chan *types.TxState, 10)

	go cmd.StartProcessor(context.TODO(), a, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "1",
		Msgs: []*types.MessageState{
			{
				MsgBody:           make([]byte, 132),
				SourceTxHash:      "1",
				SourceDomain:      0,
				DestDomain:        4,
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	cmd.State.Mu.Lock()
	defer cmd.State.Mu.Unlock()
	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Created, actualState.Msgs[0].Status)
}

// created message -> disabled cctp route -> filtered
func TestProcessDisabledCctpRoute(t *testing.T) {
	a, registeredDomains := testutil.ConfigSetup(t)

	sequenceMap := types.NewSequenceMap()
	processingQueue = make(chan *types.TxState, 10)

	go cmd.StartProcessor(context.TODO(), a, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			{
				SourceTxHash:      "123",
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        5, //not configured
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	// cmd.State.Mu.Lock()
	// defer cmd.State.Mu.Unlock()
	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Msgs[0].Status)
}

// created message -> different destination caller -> filtered
func TestProcessInvalidDestinationCaller(t *testing.T) {
	a, registeredDomains := testutil.ConfigSetup(t)

	sequenceMap := types.NewSequenceMap()
	processingQueue = make(chan *types.TxState, 10)

	go cmd.StartProcessor(context.TODO(), a, registeredDomains, processingQueue, sequenceMap)

	nonEmptyBytes := make([]byte, 31)
	nonEmptyBytes = append(nonEmptyBytes, 0x1)

	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			{
				SourceTxHash:      "123",
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        4,
				DestinationCaller: nonEmptyBytes,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	cmd.State.Mu.Lock()
	defer cmd.State.Mu.Unlock()
	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Msgs[0].Status)
}

// we want to filter out the transaction if the route is not enalbed
func TestFilterDisabledCCTPRoutes(t *testing.T) {

	logger := log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))

	var msgState types.MessageState

	cfg := types.Config{
		EnabledRoutes: map[types.Domain][]types.Domain{
			0: {1, 2},
		},
	}

	// test enabled dest domain
	msgState = types.MessageState{
		SourceDomain: types.Domain(0),
		DestDomain:   types.Domain(1),
	}
	filterTx := cmd.FilterDisabledCCTPRoutes(&cfg, logger, &msgState)
	require.False(t, filterTx)

	// test NOT enabled dest domain
	msgState = types.MessageState{
		SourceDomain: types.Domain(0),
		DestDomain:   types.Domain(3),
	}
	filterTx = cmd.FilterDisabledCCTPRoutes(&cfg, logger, &msgState)
	require.True(t, filterTx)

	// test NOT enabled source domain
	msgState = types.MessageState{
		SourceDomain: types.Domain(3),
		DestDomain:   types.Domain(1),
	}
	filterTx = cmd.FilterDisabledCCTPRoutes(&cfg, logger, &msgState)
	require.True(t, filterTx)

}

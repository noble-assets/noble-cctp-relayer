package cmd_test

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
var sequenceMap *types.SequenceMap

func setupTest(t *testing.T) map[types.Domain]types.Chain {
	var err error
	cfg, err = cmd.Parse("../.ignore/testnet.yaml")
	require.NoError(t, err, "Error parsing config")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))
	processingQueue = make(chan *types.TxState, 10000)

	n, err := cfg.Chains["noble"].(*noble.ChainConfig).Chain("noble")
	require.NoError(t, err, "Error creating noble chain")

	_, nextMinterSequence, err := n.(*noble.Noble).AccountInfo(context.TODO())
	require.NoError(t, err, "Error retrieving account sequence")

	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(types.Domain(4), nextMinterSequence)

	registeredDomains := make(map[types.Domain]types.Chain)
	for name, cfgg := range cfg.Chains {
		c, err := cfgg.Chain(name)
		require.NoError(t, err, "Error creating chain")

		registeredDomains[c.Domain()] = c
	}

	return registeredDomains

}

// new log -> create state entry
func TestProcessNewLog(t *testing.T) {
	registeredDomains := setupTest(t)

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "1",
		Msgs: []*types.MessageState{
			&types.MessageState{
				SourceTxHash:      "1",
				SourceDomain:      0,
				DestDomain:        4,
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	actualState, _ := cmd.State.Load(expectedState.TxHash)

	require.Equal(t, types.Created, actualState.Msgs[0].Status)
}

// created message -> check attestation -> mark as attested -> mark as complete -> remove from state
func TestProcessCreatedLog(t *testing.T) {
	registeredDomains := setupTest(t)
	cfg.EnabledRoutes[0] = []types.Domain{5} // skip mint

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)

	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			&types.MessageState{
				SourceTxHash:      "123",
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        5,
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Complete, actualState.Msgs[0].Status)
}

// created message -> disabled cctp route -> filtered
func TestProcessDisabledCctpRoute(t *testing.T) {
	registeredDomains := setupTest(t)

	delete(cfg.EnabledRoutes, 0)

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			{
				SourceTxHash:      "123",
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        5,
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Msgs[0].Status)
}

// created message -> different destination caller -> filtered
func TestProcessInvalidDestinationCaller(t *testing.T) {
	registeredDomains := setupTest(t)

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

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

	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Msgs[0].Status)
}

// created message -> not \ -> filtered
func TestProcessNonBurnMessageWhenDisabled(t *testing.T) {
	registeredDomains := setupTest(t)

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			{
				SourceTxHash:      "123",
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        4,
				DestinationCaller: emptyBz,
			},
		},
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Msgs[0].Status)
}

// test batch transactions where multiple messages can be sent with the same tx hash
// MsgSentBytes defer between messages
func TestBatchTx(t *testing.T) {
	registeredDomains := setupTest(t)

	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.TxState{
		TxHash: "123",
		Msgs: []*types.MessageState{
			{
				SourceTxHash: "123",
				IrisLookupId: "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:       types.Created,
				SourceDomain: 0,
				DestDomain:   4,
				MsgSentBytes: []byte("mock bytes 1"), // different message sent bytes
			},
			{
				SourceTxHash:      "123", // same source tx hash
				IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
				Status:            types.Created,
				SourceDomain:      0,
				DestDomain:        4,
				DestinationCaller: emptyBz,
				MsgSentBytes:      []byte("mock bytes 2"), // different message sent bytes
			},
		},
	}

	processingQueue <- expectedState

	actualState, ok := cmd.State.Load(expectedState.TxHash)
	require.True(t, ok)
	require.Equal(t, 2, len(actualState.Msgs))
}

// we want to filter out the transaction if the route is not enalbed
func TestFilterDisabledCCTPRoutes(t *testing.T) {

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))

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

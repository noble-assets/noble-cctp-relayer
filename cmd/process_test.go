package cmd_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

var cfg config.Config
var logger log.Logger
var processingQueue chan *types.MessageState
var sequenceMap *types.SequenceMap

func setupTest() {
	cfg = config.Parse("../.ignore/unit_tests.yaml")
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))
	processingQueue = make(chan *types.MessageState, 10000)

	_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
		cfg.Networks.Destination.Noble.API,
		cfg.Networks.Minters[4].MinterAddress)

	if err != nil {
		logger.Error("Error retrieving account sequence", "err: ", err)
		os.Exit(1)
	}
	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(4), nextMinterSequence)

}

// new log -> create state entry
func TestProcessNewLog(t *testing.T) {
	setupTest()

	p := cmd.Processor{}

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "1",
		Type:              types.Mint,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	actualState, _ := cmd.State.Load(expectedState.SourceTxHash)

	p.Mu.RLock()
	require.Equal(t, types.Created, actualState[0].Status)
	p.Mu.RUnlock()

}

// created message -> check attestation -> mark as attested -> mark as complete -> remove from state
func TestProcessCreatedLog(t *testing.T) {
	setupTest()
	cfg.Networks.EnabledRoutes[0] = 5 // skip mint

	p := cmd.NewProcessor()

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "123",
		Type:              types.Mint,
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        5,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, types.Complete, actualState[0].Status)
	p.Mu.RUnlock()
}

// created message -> disabled cctp route -> filtered
func TestProcessDisabledCctpRoute(t *testing.T) {
	setupTest()

	delete(cfg.Networks.EnabledRoutes, 0)

	p := cmd.NewProcessor()

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "123",
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        5,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, types.Filtered, actualState[0].Status)
	p.Mu.RUnlock()
}

// created message -> different destination caller -> filtered
func TestProcessInvalidDestinationCaller(t *testing.T) {
	setupTest()

	p := cmd.NewProcessor()

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	nonEmptyBytes := make([]byte, 31)
	nonEmptyBytes = append(nonEmptyBytes, 0x1)

	expectedState := &types.MessageState{
		SourceTxHash:      "123",
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: nonEmptyBytes,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, types.Filtered, actualState[0].Status)
	p.Mu.RUnlock()
}

// created message -> nonwhitelisted channel -> filtered
func TestProcessNonWhitelistedChannel(t *testing.T) {
	setupTest()
	cfg.Networks.Destination.Noble.FilterForwardsByIbcChannel = true

	p := cmd.NewProcessor()

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "123",
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, types.Filtered, actualState[0].Status)
	p.Mu.RUnlock()
}

// created message -> not \ -> filtered
func TestProcessNonBurnMessageWhenDisabled(t *testing.T) {
	setupTest()

	p := cmd.NewProcessor()

	go p.StartProcessor(cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "123",
		Type:              "",
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, types.Filtered, actualState[0].Status)
	p.Mu.RUnlock()

}

// test batch transactions where multiple messages can be sent with the same tx hash
// MsgSentBytes defer between messages
func TestBatchTx(t *testing.T) {
	setupTest()

	p := cmd.NewProcessor()

	go p.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "123", // same source tx hash
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
		MsgSentBytes:      []byte("mock bytes 1"), // different message sent bytes
	}
	processingQueue <- expectedState

	expectedState2 := &types.MessageState{
		SourceTxHash:      "123", // same source tx hash
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
		MsgSentBytes:      []byte("mock bytes 2"), // different message sent bytes
	}

	processingQueue <- expectedState2
	time.Sleep(6 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.SourceTxHash)
	require.True(t, ok)
	p.Mu.RLock()
	require.Equal(t, 2, len(actualState))
	p.Mu.RUnlock()
}

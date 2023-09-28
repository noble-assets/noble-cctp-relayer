package cmd_test

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
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
var sequenceMap *types.SequenceMap

func setupTest() {
	cfg = config.Parse("../.ignore/unit_tests.yaml")
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.Disabled))
	processingQueue = make(chan *types.MessageState, 10000)

	_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
		cfg.Networks.Destination.Noble.API,
		cfg.Networks.Minters[4].MinterAddress)

	if err != nil {
		logger.Error("Error retrieving account sequence")
		os.Exit(1)
	}
	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(4), nextMinterSequence)

}

// new log -> create state entry
func TestProcessNewLog(t *testing.T) {
	setupTest()

	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		SourceTxHash:      "1",
		Type:              types.Mint,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, _ := cmd.State.Load(cmd.LookupKey(expectedState.SourceTxHash, expectedState.Type))

	require.Equal(t, types.Created, actualState.Status)

}

// created message -> check attestation -> mark as attested -> mark as complete -> remove from state
func TestProcessCreatedLog(t *testing.T) {
	setupTest()
	cfg.Networks.EnabledRoutes[0] = 5 // skip mint

	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

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

	actualState, ok := cmd.State.Load(cmd.LookupKey(expectedState.SourceTxHash, expectedState.Type))
	require.True(t, ok)
	require.Equal(t, types.Complete, actualState.Status)

}

// created message -> disabled cctp route -> filtered
func TestProcessDisabledCctpRoute(t *testing.T) {
	setupTest()

	delete(cfg.Networks.EnabledRoutes, 0)

	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

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

	actualState, ok := cmd.State.Load(cmd.LookupKey(expectedState.SourceTxHash, expectedState.Type))
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

// created message -> different destination caller -> filtered
func TestProcessInvalidDestinationCaller(t *testing.T) {
	setupTest()

	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

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

	actualState, ok := cmd.State.Load(cmd.LookupKey(expectedState.SourceTxHash, expectedState.Type))
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

// created message -> nonwhitelisted channel -> filtered
func TestProcessNonWhitelistedChannel(t *testing.T) {
	setupTest()
	cfg.Networks.Destination.Noble.FilterForwardsByIbcChannel = true

	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

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

	actualState, ok := cmd.State.Load(cmd.LookupKey(expectedState.SourceTxHash, expectedState.Type))
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

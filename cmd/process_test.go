package cmd_test

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
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

func setupTest() {
	cfg = config.Parse("../.ignore/testing.yaml")
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

// new log -> create state entry
func TestProcessNewLog(t *testing.T) {
	setupTest()

	go cmd.StartProcessor(cfg, logger, processingQueue)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		IrisLookupId:      "1",
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	actualState, _ := cmd.State.Load(expectedState.IrisLookupId)

	require.Equal(t, types.Created, actualState.Status)

}

// created message -> check attestation -> mark as attested -> mark as complete -> remove from state
func TestProcessCreatedLog(t *testing.T) {
	setupTest()
	cfg.Networks.EnabledRoutes[0] = 5 // skip mint

	go cmd.StartProcessor(cfg, logger, processingQueue)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        5,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.IrisLookupId)
	require.True(t, ok)
	require.Equal(t, types.Complete, actualState.Status)

}

// created message -> disabled cctp route -> filtered
func TestProcessDisabledCctpRoute(t *testing.T) {
	setupTest()

	delete(cfg.Networks.EnabledRoutes, 0)

	go cmd.StartProcessor(cfg, logger, processingQueue)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        5,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.IrisLookupId)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

// created message -> different destination caller -> filtered
func TestProcessInvalidDestinationCaller(t *testing.T) {
	setupTest()

	go cmd.StartProcessor(cfg, logger, processingQueue)

	nonEmptyBytes := make([]byte, 31)
	nonEmptyBytes = append(nonEmptyBytes, 0x1)

	expectedState := &types.MessageState{
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: nonEmptyBytes,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.IrisLookupId)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

// created message -> nonwhitelisted channel -> filtered
func TestProcessNonWhitelistedChannel(t *testing.T) {
	setupTest()
	cfg.Networks.Destination.Noble.FilterForwardsByIbcChannel = true

	go cmd.StartProcessor(cfg, logger, processingQueue)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Status:            types.Created,
		SourceDomain:      0,
		DestDomain:        4,
		DestinationCaller: emptyBz,
	}

	processingQueue <- expectedState

	time.Sleep(2 * time.Second)

	actualState, ok := cmd.State.Load(expectedState.IrisLookupId)
	require.True(t, ok)
	require.Equal(t, types.Filtered, actualState.Status)

}

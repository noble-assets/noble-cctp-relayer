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

func init() {
	cfg = config.Parse("/Users/joel/src/noble-cctp-relayer/.ignore/testing.yaml")
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

func TestProcessSuccess(t *testing.T) {
	go cmd.StartProcessor(cfg, logger, processingQueue)

	emptyBz := make([]byte, 32)
	expectedState := &types.MessageState{
		//IrisLookupId:      "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		IrisLookupId:      "1",
		Type:              "",
		Status:            "",
		Attestation:       "",
		SourceDomain:      0,
		DestDomain:        4,
		SourceTxHash:      "",
		DestTxHash:        "",
		MsgSentBytes:      nil,
		DestinationCaller: emptyBz,
		Channel:           "",
		Created:           time.Time{},
		Updated:           time.Time{},
	}

	processingQueue <- expectedState

	time.Sleep(5 * time.Second)

	actualState := cmd.State[expectedState.IrisLookupId]

	require.Equal(t, types.Created, actualState.Status)

}

// process tests for each state

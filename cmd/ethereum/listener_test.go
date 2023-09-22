package ethereum_test

import (
	"cosmossdk.io/log"
	"encoding/base64"
	"fmt"
	"github.com/rs/zerolog"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
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
	cfg = config.Parse("../../.ignore/testing.yaml")

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	processingQueue = make(chan *types.MessageState, 10000)
}

func TestStartListener(t *testing.T) {

	cfg.Networks.Source.Ethereum.StartBlock = 9702735
	cfg.Networks.Source.Ethereum.LookbackPeriod = 0
	go eth.StartListener(cfg, logger, processingQueue)

	time.Sleep(5 * time.Second)

	msg := <-processingQueue

	expectedMsg := &types.MessageState{
		IrisLookupId: "a404f4155166a1fc7ffee145b5cac6d0f798333745289ab1db171344e226ef0c",
		Type:         "mint",
		Status:       "created",
		SourceDomain: 0,
		DestDomain:   4,
		SourceTxHash: "0xe1d7729de300274ee3a2fd20ba179b14a8e3ffcd9d847c506b06760f0dad7802",
	}
	require.Equal(t, expectedMsg.IrisLookupId, msg.IrisLookupId)
	require.Equal(t, expectedMsg.Type, msg.Type)
	require.Equal(t, expectedMsg.Status, msg.Status)
	require.Equal(t, expectedMsg.SourceDomain, msg.SourceDomain)
	require.Equal(t, expectedMsg.DestDomain, msg.DestDomain)
	require.Equal(t, expectedMsg.SourceTxHash, msg.SourceTxHash)

}

func TestDelete(t *testing.T) {
	msg, err := base64.StdEncoding.DecodeString("AAAAAAAAAAAAAAAEAAAAAAADlqQAAAAAAAAAAAAAAAAa4EXZkjY2XL3BhVrNLSz8Iy0E0QAAAAAAAAAAAAAAAHTJVgQENCfwvuHQ4Wv6U6/VN/c2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOWowAAAAAAAAAAAAAAAJccVKbreC+szQC8PtXpNMxb2OPvAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZHlkeAAAAAAAAAAAAAAAAAgaAk0u4iu1ufOjCF0jiUzEoOMi")
	require.NoError(t, err)

	messageState, err := new(types.Message).Parse(msg)
	require.NoError(t, err)

	burnMessage, err := new(types.BurnMessage).Parse(msg)
	fmt.Println()
	metadataMessage, err := new(types.MetadataMessage).Parse(msg)
	fmt.Println()

	t.Logf("nonce: %d", messageState.Nonce)
	fmt.Println(burnMessage)
	fmt.Println(metadataMessage)
}

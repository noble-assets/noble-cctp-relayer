package types_test

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/pascaldekloe/etherstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutil "github.com/strangelove-ventures/noble-cctp-relayer/test_util"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// TODO: update so it doesn't rely on block history
func TestToMessageStateSuccess(t *testing.T) {
	err := godotenv.Load(testutil.EnvFile)
	require.NoError(t, err)

	messageTransmitter, err := os.Open("../cmd/ethereum/abi/MessageTransmitter.json")
	require.NoError(t, err)

	messageTransmitterABI, err := abi.JSON(messageTransmitter)
	require.NoError(t, err)

	messageSent := messageTransmitterABI.Events["MessageSent"]

	ethClient, err := ethclient.DialContext(context.Background(), os.Getenv("SEPOLIA_RPC"))
	require.NoError(t, err)

	messageTransmitterAddress := common.HexToAddress("0x26413e8157CD32011E726065a5462e97dD4d03D9")

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{messageSent.ID}},
		FromBlock: big.NewInt(9573853),
		ToBlock:   big.NewInt(9573853),
	}

	etherReader := etherstream.Reader{Backend: ethClient}

	_, _, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	require.NoError(t, err)

	messageState, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &history[0])

	event := make(map[string]interface{})
	_ = messageTransmitterABI.UnpackIntoMap(event, messageSent.Name, history[0].Data)

	rawMessageSentBytes := event["message"].([]byte)

	destCaller := make([]byte, 32)
	assert.Equal(t, "e40ed0e983675678715972bd50d6abc417735051b0255f3c0916911957eda603", messageState.IrisLookupID)
	assert.Equal(t, "created", messageState.Status)
	assert.Equal(t, "", messageState.Attestation)
	assert.Equal(t, uint32(0), messageState.SourceDomain)
	assert.Equal(t, uint32(4), messageState.DestDomain)
	assert.Equal(t, "0xed567f5a62166d0a5df6cdcec710640b1c8079758cd1e1ac95085742f06afb04", messageState.SourceTxHash)
	assert.Equal(t, "", messageState.DestTxHash)
	assert.Equal(t, rawMessageSentBytes, messageState.MsgSentBytes)
	assert.Equal(t, destCaller, messageState.DestinationCaller)
	assert.Equal(t, "", messageState.Channel)
	t.Log(messageState)
	require.NoError(t, err)
}

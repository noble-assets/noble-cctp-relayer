package types_test

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pascaldekloe/etherstream"
	"github.com/stretchr/testify/require"
	"math/big"
	"os"
	"testing"
)

var eth_rpc = "wss://goerli.infura.io/ws/v3/d33a2875a8e6483da6ce10fe607eba31"

func TestToMessageStateSuccess(t *testing.T) {

	messageTransmitter, err := os.Open("../abi/MessageTransmitter.json")
	require.Nil(t, err)

	messageTransmitterABI, err := abi.JSON(messageTransmitter)
	require.Nil(t, err)

	messageSent := messageTransmitterABI.Events["MessageSent"]

	ethClient, err := ethclient.DialContext(context.Background(), eth_rpc)
	require.Nil(t, err)

	messageTransmitterAddress := common.HexToAddress("0x26413e8157CD32011E726065a5462e97dD4d03D9")

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{messageSent.ID}},
		FromBlock: big.NewInt(9573853),
		ToBlock:   big.NewInt(9573853),
	}

	etherReader := etherstream.Reader{Backend: ethClient}

	_, _, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	require.Nil(t, err)

	fmt.Println(history)

	// TODO
	//messageState, err := ToMessageState(messageTransmitterABI, messageSent)
	//require.Nil(t, err)
}

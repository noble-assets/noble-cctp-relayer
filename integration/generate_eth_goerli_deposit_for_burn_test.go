package integration_testing

import (
	"context"
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"io"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"
)

var testCfg Config
var mintCfg config.Config
var logger log.Logger

// goerli
const TokenMessengerAddress = "0xd0c3da58f55358142b8d3e06c1c30c5c6114efe8"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"
const NobleAddress = "noble1wa5g4at8yfmph96jxsvn0ynnf5qx73h0l6ecrs"

func setupTest() func() {
	// setup
	testCfg = Parse("../.ignore/integration.yaml")
	//mintCfg = config.Parse("") // TODO
	logger = log.NewLogger(os.Stdout)

	return func() {
		// tear-down
	}
}

// TestGenerateEthDepositForBurn generates and broadcasts a depositForBurn on Ethereum Goerli
func TestGenerateEthDepositForBurn(t *testing.T) {
	setupTest()

	// client
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(testCfg.Networks.Ethereum.PrivateKey)
	require.Nil(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	require.Nil(t, err)
	// set nonce
	//nextNonce, err := client.PendingNonceAt(context.Background(), auth.From)
	//require.Nil(t, err)
	//auth.Nonce = big.NewInt(int64(nextNonce))

	tokenMessenger, err := cmd.NewTokenMessenger(common.HexToAddress(TokenMessengerAddress), client)
	require.Nil(t, err)

	_, mintRecipientBz, _ := bech32.DecodeAndConvert(NobleAddress)
	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, mintRecipientBz...)
	require.Nil(t, err)

	// approve amount
	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddress), big.NewInt(99999))
	require.Nil(t, err)

	tx, err := tokenMessenger.DepositForBurn(
		auth,
		big.NewInt(21),
		4,
		[32]byte(mintRecipientPadded),
		common.HexToAddress(UsdcAddress),
	)
	if err != nil {
		logger.Error("Failed to update value: %v", err)
	}
	fmt.Printf("Update pending: https://goerli.etherscan.io/tx/0x%x\n", tx.Hash())
	messageTransmitterAddress := common.HexToAddress("0x26413e8157CD32011E726065a5462e97dD4d03D9")
	messageTransmitter, err := os.Open("../abi/MessageTransmitter.json")
	require.Nil(t, err)

	messageTransmitterAbi, err := abi.JSON(messageTransmitter)
	require.Nil(t, err)
	messageSent := messageTransmitterAbi.Events["MessageSent"]
	require.Nil(t, err)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(
			9702611),
		Addresses: []common.Address{
			messageTransmitterAddress,
		},
		Topics: [][]common.Hash{{messageSent.ID}},
	}

	var parsedMsg *types.MessageState
	for i := 0; i < 5; i++ {
		logs, err := client.FilterLogs(context.Background(), query)
		require.Nil(t, err)

		for _, log := range logs {
			parsedMsg, err = types.ToMessageState(messageTransmitterAbi, messageSent, &log)
			if err == nil && parsedMsg.SourceTxHash == tx.Hash().Hex() {
				fmt.Printf("Attestation url: https://iris-api-sandbox.circle.com/attestations/0x%s\n", parsedMsg.IrisLookupId)
				goto Exit
			}
		}
		fmt.Println("No logs found.  Retrying...")
		time.Sleep(10 * time.Second)
	}
	fmt.Println("No logs found after 5 retries. Exiting...")
	os.Exit(1)
Exit:

	fmt.Println("Waiting 1 minute for the attestation to finalize...")
	time.Sleep(1 * time.Minute)
	for i := 0; i < 20; i++ {

		rawResponse, err := http.Get("https://iris-api-sandbox.circle.com/attestations/0x" + parsedMsg.IrisLookupId)
		require.Nil(t, err)

		body, err := io.ReadAll(rawResponse.Body)
		require.Nil(t, err)

		response := types.AttestationResponse{}
		err = json.Unmarshal(body, &response)

		if response.Status == "complete" {
			parsedMsg.Attestation = response.Attestation
			break
		}

		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}

	txResponse, err := noble.Broadcast(mintCfg, logger, *parsedMsg)
	if err != nil {
		fmt.Println("Error during broadcast: " + err.Error())
		os.Exit(1)
	}

	require.Equal(t, uint32(0), txResponse.Code)
	fmt.Printf("Minted at https://testnet.mintscan.io/noble-testnet/txs/%s\n", txResponse.TxHash)

	fmt.Print("Finished")
}

package integration_testing

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	cmd "github.com/strangelove-ventures/noble-cctp-relayer/abi"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"log"
	"math/big"
	"os"
	"testing"
	"time"
)

var cfg Config

// goerli
const TokenMessengerAddress = "0xd0c3da58f55358142b8d3e06c1c30c5c6114efe8"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"
const NobleAddress = "noble1wa5g4at8yfmph96jxsvn0ynnf5qx73h0l6ecrs"

func setupTest() func() {
	// setup
	cfg = Parse("../.ignore/integration.yaml")

	return func() {
		// tear-down
	}
}

// TestGenerateEthDepositForBurn generates and broadcasts a depositForBurn on Ethereum Goerli
func TestGenerateEthDepositForBurn(t *testing.T) {
	setupTest()

	// client
	client, err := ethclient.Dial(cfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(cfg.Networks.Ethereum.PrivateKey)
	require.Nil(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	require.Nil(t, err)

	tokenMessenger, err := cmd.NewTokenMessenger(common.HexToAddress(TokenMessengerAddress), client)
	require.Nil(t, err)

	_, mintRecipientBz, _ := bech32.DecodeAndConvert(NobleAddress)
	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, mintRecipientBz...)
	require.Nil(t, err)

	// approve amount
	erc20, err := cmd.NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddress), big.NewInt(99999))
	require.Nil(t, err)

	tx, err := tokenMessenger.DepositForBurn(
		auth,
		big.NewInt(20000), // 0.02
		4,
		[32]byte(mintRecipientPadded),
		common.HexToAddress(UsdcAddress),
	)
	if err != nil {
		log.Fatalf("Failed to update value: %v", err)
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
		FromBlock: big.NewInt(9695615),
		Addresses: []common.Address{
			messageTransmitterAddress,
		},
		Topics: [][]common.Hash{{messageSent.ID}},
	}

	for i := 0; i < 5; i++ {
		logs, err := client.FilterLogs(context.Background(), query)
		require.Nil(t, err)

		for _, log := range logs {
			parsedMsg, err := types.ToMessageState(messageTransmitterAbi, messageSent, &log)
			if err == nil && parsedMsg.SourceDomain == 0 && parsedMsg.DestDomain == 4 {
				fmt.Printf("Attestation url: https://iris-api-sandbox.circle.com/attestations/0x%s\n", parsedMsg.IrisLookupId)
				break
			}
		}
		fmt.Println("No logs found.  Retrying...")
		time.Sleep(10 * time.Second)
	}
	fmt.Print("Finished")
}

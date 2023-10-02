package integration_testing

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"log"
	"math/big"
	"strings"
	"testing"
	"time"
)

// TestNobleBurnToEthMint generates and broadcasts a depositForBurn on Noble
// and broadcasts on Ethereum Goerli
func TestNobleBurnToEthMint(t *testing.T) {
	setupTest()

	// start up relayer
	cfg.Networks.Source.Noble.StartBlock = getNobleLatestBlockHeight()
	cfg.Networks.Source.Noble.LookbackPeriod = 0

	fmt.Println("Starting relayer...")
	processingQueue := make(chan *types.MessageState, 10)
	go noble.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

	fmt.Println("Building Noble depositForBurn txn...")
	ethAddress := "0x971c54a6Eb782fAccD00bc3Ed5E934Cc5bD8e3Ef"
	fmt.Println("Minting on Ethereum to https://goerli.etherscan.io/address/" + ethAddress)

	// verify ethereum usdc amount
	client, _ := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	defer client.Close()
	originalEthBalance := getEthBalance(client, ethAddress)

	// deposit for burn
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	var burnAmount = big.NewInt(1)

	// TODO sample deposit for burn noble

	time.Sleep(5 * time.Second)
	//fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

	fmt.Println("Checking eth wallet...")
	for i := 0; i < 60; i++ {
		if originalEthBalance+burnAmount.Uint64() == getEthBalance(client, ethAddress) {
			fmt.Println("Successfully minted at https://goerli.etherscan.io/address/" + ethAddress)
			return
		}
		time.Sleep(1 * time.Second)
	}
	// verify eth balance
	require.Equal(t, originalEthBalance+burnAmount.Uint64(), getEthBalance(client, ethAddress))
}

func getEthBalance(client *ethclient.Client, address string) uint64 {
	accountAddress := common.HexToAddress(address)
	tokenAddress := common.HexToAddress("0x07865c6e87b9f70255377e024ace6630c1eaa37f") // USDC goerli
	erc20ABI := `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}

	data, err := parsedABI.Pack("balanceOf", accountAddress)
	if err != nil {
		log.Fatalf("Failed to pack data into ABI interface: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{To: &tokenAddress, Data: data}, nil)
	if err != nil {
		log.Fatalf("Failed to call contract: %v", err)
	}

	balance := new(big.Int)
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		log.Fatalf("Failed to unpack data from ABI interface: %v", err)
	}

	// Convert to uint64
	return balance.Uint64()
}

package integration_testing

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

// TestNobleMultiSend broadcasts N messages on Ethereum, and then tries to receive them all at once on Noble.
// The point of this test is to verify that account sequences on Noble are synced correctly.
// A success means that all messages went through (i.e. no retries are attempted).
func TestNobleMultiSend(t *testing.T) {
	setupTest()

	// number of depositForBurn txns to send
	n := 5

	// start up relayer
	cfg.Networks.Source.Ethereum.StartBlock = getEthereumLatestBlockHeight(t)
	cfg.Networks.Source.Ethereum.LookbackPeriod = 20
	cfg.Networks.Destination.Noble.BroadcastRetries = 0

	fmt.Println(fmt.Sprintf("Building %d Ethereum depositForBurnWithMetadata txn..."), n)

	_, _, cosmosAddress := testdata.KeyTestPubAddr()
	nobleAddress, _ := bech32.ConvertAndEncode("noble", cosmosAddress)
	fmt.Println("Minting on Noble to https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)

	// verify original noble usdc amount
	originalNobleBalance := getNobleBalance(nobleAddress)

	// deposit for burn with metadata
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(testCfg.Networks.Ethereum.PrivateKey)
	require.Nil(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	require.Nil(t, err)

	tokenMessenger, err := cmd.NewTokenMessenger(common.HexToAddress(TokenMessengerAddress), client)
	require.Nil(t, err)

	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cosmosAddress...)
	require.Nil(t, err)

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerWithMetadataAddress), big.NewInt(99999))
	require.Nil(t, err)
	var burnAmount = big.NewInt(1)

	for i := 0; i < n; i++ {
		tx, err := tokenMessenger.DepositForBurn(
			auth,
			burnAmount,
			4,
			[32]byte(mintRecipientPadded),
			common.HexToAddress(UsdcAddress),
		)
		if err != nil {
			logger.Error("Failed to update value: %v", err)
		}

		fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

		// todo sleep between burns?
	}

	fmt.Println("Waiting 90 seconds for attestations...")
	time.Sleep(90 * time.Second)

	fmt.Println("Starting relayer...")
	processingQueue := make(chan *types.MessageState, 100)

	go eth.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

	fmt.Println("Checking noble wallet...")
	for i := 0; i < 250; i++ {
		if originalNobleBalance+burnAmount.Uint64()*uint64(n) == getNobleBalance(nobleAddress) {
			fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)
			return
		}
		time.Sleep(1 * time.Second)
	}
	// verify noble balance
	require.Equal(t, originalNobleBalance+burnAmount.Uint64()*uint64(n), getNobleBalance(nobleAddress))

	// check logs

}

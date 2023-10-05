package integration_testing

import (
	"encoding/hex"
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

// TestEthBurnToNobleMintAndForward generates a depositForBurn on Ethereum Goerli and mints + forwards on Noble
func TestEthBurnToNobleMintAndForward(t *testing.T) {
	setupTest()

	// start up relayer
	cfg.Networks.Source.Ethereum.StartBlock = getEthereumLatestBlockHeight(t)
	cfg.Networks.Source.Ethereum.LookbackPeriod = 0

	fmt.Println("Starting relayer...")
	processingQueue := make(chan *types.MessageState, 10)
	go eth.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(cfg, logger, processingQueue, sequenceMap)

	fmt.Println("Building Ethereum depositForBurnWithMetadata txn...")
	_, _, cosmosAddress := testdata.KeyTestPubAddr()
	nobleAddress, _ := bech32.ConvertAndEncode("noble", cosmosAddress)
	fmt.Println("Intermediately minting on Noble to " + nobleAddress)

	_, _, cosmosAddress2 := testdata.KeyTestPubAddr()
	dydxAddress, _ := bech32.ConvertAndEncode("dydx", cosmosAddress2)
	fmt.Println("Forwarding funds to " + dydxAddress)

	// verify dydx usdc amount
	originalDydx := getDydxBalance(dydxAddress)

	// deposit for burn with metadata
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(testCfg.Networks.Ethereum.PrivateKey)
	require.Nil(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	require.Nil(t, err)

	tokenMessengerWithMetadata, err := cmd.NewTokenMessengerWithMetadata(common.HexToAddress(TokenMessengerWithMetadataAddress), client)
	require.Nil(t, err)

	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cosmosAddress...)
	require.Nil(t, err)

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerWithMetadataAddress), big.NewInt(99999))
	require.Nil(t, err)

	channel := uint64(20)
	destinationBech32Prefix :=
		append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, []byte("dydx")...)
	destinationRecipient := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cosmosAddress2...)

	var BurnAmount = big.NewInt(1)

	f := hex.EncodeToString(mintRecipientPadded)
	fmt.Println(f)

	tx, err := tokenMessengerWithMetadata.DepositForBurn(
		auth,
		channel,                           // channel
		[32]byte(destinationBech32Prefix), // destinationBech32Prefix
		[32]byte(destinationRecipient),    // destinationRecipient
		BurnAmount,                        // amount
		[32]byte(mintRecipientPadded),     // mint recipient
		common.HexToAddress(UsdcAddress),  // burn token
		[]byte{},                          // memo
	)
	if err != nil {
		logger.Error("Failed to update value: %v", err)
	}

	time.Sleep(5 * time.Second)
	fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

	fmt.Println("Checking dydx wallet...")
	for i := 0; i < 250; i++ {
		if originalDydx+BurnAmount.Uint64() == getDydxBalance(dydxAddress) {
			fmt.Println("Successfully minted at https://testnet.mintscan.io/dydx-testnet/account/" + dydxAddress)
			return
		}
		time.Sleep(1 * time.Second)
	}
	// verify dydx balance
	require.Equal(t, originalDydx+BurnAmount.Uint64(), getDydxBalance(dydxAddress))
}

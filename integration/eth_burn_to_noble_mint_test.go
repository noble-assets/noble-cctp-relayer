package integration_testing

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

const uusdcDenom = "uusdc"

// TestEthBurnToNobleMint generates a depositForBurn on Ethereum Goerli and mints on Noble
func TestEthBurnToNobleMint(t *testing.T) {
	ctx := context.Background()
	setupTestIntegration()

	fmt.Println("Starting relayer...")
	processingQueue := make(chan *types.TxState, 10)

	registeredDomains := make(map[types.Domain]types.Chain)
	registeredDomains[0] = ethChain
	registeredDomains[4] = nobleChain

	nobleChain.InitializeBroadcaster(ctx, logger, sequenceMap)

	go ethChain.StartListener(ctx, logger, processingQueue)
	go cmd.StartProcessor(ctx, cfg, logger, registeredDomains, processingQueue, sequenceMap)

	fmt.Println("Building Ethereum depositForBurn txn...")
	_, _, cosmosAddress := testdata.KeyTestPubAddr()
	nobleAddress, _ := bech32.ConvertAndEncode("noble", cosmosAddress)
	fmt.Println("Minting on Noble to https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)

	// verify noble usdc amount
	cc, err := cosmos.NewProvider(nobleCfg.RPC)
	require.Nil(t, err)
	// originalNobleBalance := getNobleBalance(nobleAddress)
	originalNobleBalance, err := getNobleAccountBalance(ctx, cc, nobleAddress, uusdcDenom)
	require.NoError(t, err)

	// deposit for burn with metadata
	client, err := ethclient.Dial(ethCfg.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(integrationWallets.Networks.Ethereum.PrivateKey)

	require.Nil(t, err)

	sepoliaChainID := big.NewInt(ethCfg.ChainID)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, sepoliaChainID)
	require.Nil(t, err)

	tokenMessenger, err := contracts.NewTokenMessenger(common.HexToAddress(TokenMessengerAddress), client)
	require.Nil(t, err)

	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cosmosAddress...)
	require.Nil(t, err)

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	require.NoError(t, err)

	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddress), big.NewInt(99999))
	require.Nil(t, err)

	var burnAmount = big.NewInt(1)

	tx, err := tokenMessenger.DepositForBurn(
		auth,
		burnAmount,
		4,
		[32]byte(mintRecipientPadded),
		common.HexToAddress(UsdcAddress),
	)
	if err != nil {
		logger.Error("Failed to update value", "err", err)
	}

	time.Sleep(5 * time.Second)
	fmt.Printf("Update pending: https://sepolia.etherscan.io/tx/%s\n", tx.Hash().String())

	var newBalance uint64
	fmt.Println("Checking noble wallet...")
	for i := 0; i < 250; i++ {
		newBalance, err = getNobleAccountBalance(ctx, cc, nobleAddress, uusdcDenom)
		require.NoError(t, err)
		if originalNobleBalance+burnAmount.Uint64() == newBalance {
			fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)
			return
		}
		time.Sleep(1 * time.Second)
	}
	// verify noble balance
	require.Equal(t, originalNobleBalance+burnAmount.Uint64(), newBalance)
}

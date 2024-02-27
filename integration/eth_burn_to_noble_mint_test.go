package integration_testing

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

// TestEthBurnToNobleMint broadcasts a depositForBurn on Sepolia of 0.000001 USDC
// to a randomly generated grand-1 (noble) wallet
//
// Requires:
//   - Config is setup and config file is passed into cmd.Parse
//   - Config must have a chain named "noble" (grand-1 testnet) and another named "sepolia"
//   - must have at least  0.000001 USDC in Sepolia relayer wallet
//
// This test uses the Noble wallet in the config as the destination caller to ensure that
// to ensure that this instance of the relayer picks up the transaction
func TestEthBurnToNobleMint(t *testing.T) {
	a := cmd.NewappState()
	a.Debug = true
	a.InitLogger()

	ctx := context.Background()

	// Relayer config
	cfg, err := cmd.ParseConfig("../.ignore/testnet.yaml")
	require.NoError(t, err)
	a.Config = cfg

	nobleCfg := a.Config.Chains["noble"].(*noble.ChainConfig)
	ethCfg := a.Config.Chains["sepolia"].(*ethereum.ChainConfig)

	nobleChain, err := nobleCfg.Chain("noble")
	require.NoError(t, err)
	ethChain, err := ethCfg.Chain("eth")
	require.NoError(t, err)

	var burnAmount = big.NewInt(1)

	fmt.Println("Starting relayer...")
	registeredDomains := make(map[types.Domain]types.Chain)
	registeredDomains[0] = ethChain
	registeredDomains[4] = nobleChain

	sequenceMap := types.NewSequenceMap()
	err = nobleChain.InitializeBroadcaster(ctx, a.Logger, sequenceMap)
	require.NoError(t, err)

	processingQueue := make(chan *types.TxState, 10)

	go ethChain.StartListener(ctx, a.Logger, processingQueue)
	go cmd.StartProcessor(ctx, a, registeredDomains, processingQueue, sequenceMap)

	_, _, generatedWallet := testdata.KeyTestPubAddr()
	destAddress, _ := bech32.ConvertAndEncode("noble", generatedWallet)
	fmt.Println("Noble destination address: ", destAddress)
	fmt.Println("Minting on Noble to https://testnet.mintscan.io/noble-testnet/account/" + destAddress)

	// verify noble usdc amount
	cc, err := cosmos.NewProvider(nobleCfg.RPC)
	require.Nil(t, err)
	originalNobleBalance, err := getNobleAccountBalance(ctx, cc, destAddress, uusdcDenom)
	require.NoError(t, err)

	// eth client
	client, err := ethclient.Dial(ethCfg.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(ethCfg.MinterPrivateKey)
	require.Nil(t, err)

	sepoliaChainID := big.NewInt(ethCfg.ChainID)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, sepoliaChainID)
	require.Nil(t, err)

	// deal w/ nonce
	ethRelayerAddress, err := ethConvertPrivateKeytoAddress(ethCfg.MinterPrivateKey)
	require.NoError(t, err)
	nextNonce, err := ethereum.GetEthereumAccountNonce(ethCfg.RPC, ethRelayerAddress)
	require.NoError(t, err)
	auth.Nonce = big.NewInt(nextNonce)

	// Approve erc20 to interact with contract up to the sum of the amount being burnt
	erc20, err := NewERC20(common.HexToAddress(usdcTokenAddressSepolia), client)
	require.NoError(t, err)

	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddressSepolia), burnAmount)
	require.Nil(t, err)

	// Ensure approval is on chain
	time.Sleep(20 * time.Second)

	// create tokenMessenger
	tokenMessenger, err := contracts.NewTokenMessenger(common.HexToAddress(TokenMessengerAddressSepolia), client)
	require.Nil(t, err)

	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, generatedWallet...)
	require.Nil(t, err)

	auth.Nonce = big.NewInt(nextNonce + 1)

	// destination caller
	callerPrivKey := nobleCfg.MinterPrivateKey
	keyBz, err := hex.DecodeString(callerPrivKey)
	require.NoError(t, err)
	privKey := secp256k1.PrivKey{Key: keyBz}
	caller, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.Nil(t, err)
	_, callerRaw, err := bech32.DecodeAndConvert(caller)
	require.NoError(t, err)
	destinationCallerPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, callerRaw...)

	tx, err := tokenMessenger.DepositForBurnWithCaller(
		auth,
		burnAmount,
		4,
		[32]byte(mintRecipientPadded),
		common.HexToAddress(usdcTokenAddressSepolia),
		[32]byte(destinationCallerPadded),
	)
	if err != nil {
		a.Logger.Error("Failed to update value", "err", err)
	}

	time.Sleep(5 * time.Second)
	fmt.Printf("Transaction broadcasted: https://sepolia.etherscan.io/tx/%s\n", tx.Hash().String())

	var newBalance uint64
	fmt.Println("Waiting for circle to approve and destination wallet to receive funds.")
	for i := 0; i < 250; i++ {
		newBalance, err = getNobleAccountBalance(ctx, cc, destAddress, uusdcDenom)
		require.NoError(t, err)
		if originalNobleBalance+burnAmount.Uint64() == newBalance {
			fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + destAddress)
			break
		}
		time.Sleep(2 * time.Second)
	}
	// verify noble balance
	require.Equal(t, originalNobleBalance+burnAmount.Uint64(), newBalance)
}

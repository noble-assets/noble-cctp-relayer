package integration_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"
	"time"

	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"cosmossdk.io/math"

	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
)

// The tests in this file are meant to test an actively deployed relayer.
//
// This test broadcasts a tx from Noble to Eth/L2 OR visa versa.
// Uncomment/comment the desired Network and Chain at the top the function.
//
// It uses the Destination Caller to ensure the specified relayer picks up the transaction.
//
// You must have a ingetartion specific config file set up. See config/sample-integration-config.yaml an example.
// This file must be passed into "ParseIntegration" at the top of the test.

// Test noble burn to Eth deposit on actively deployed relayer
func TestNobleBurnToEthDeployed(t *testing.T) {
	c, err := ParseIntegration("../.ignore/integration.yaml")
	require.NoError(t, err)
	ctx := context.Background()

	// -- NETWORK --
	cfg := c.Testnet
	// cfg := c.Mainnet

	// -- SRC CHAIN --
	nobleCfg := cfg["noble"]

	// -- DEST CHAIN --
	ethConfig := cfg["ethereum"]
	// ethConfig := cfg["arbitrum"]
	// ethConfig := cfg["avalanche"]
	// ethConfig := cfg["optimism"]
	// ethConfig := cfg["polygon"]
	// ethConfig := cfg["base"]

	destAddress := ethConfig.Address

	var burnAmount = math.NewInt(1)

	t.Logf("Path: %d -> %d", nobleCfg.Domain, ethConfig.Domain)

	t.Logf("Source Address: %s", nobleCfg.Address)
	cc, err := cosmos.NewProvider(nobleCfg.RPC)
	require.NoError(t, err)
	originalNobleBalance, err := getNobleAccountBalance(ctx, cc, nobleCfg.Address, uusdcDenom)
	require.NoError(t, err)
	t.Logf("Source Balance: %d", originalNobleBalance)

	t.Logf("Deposit Address: %s", destAddress)

	// Get original usdc balance on Eth to verify that funds are deposited later
	client, _ := ethclient.Dial(ethConfig.RPC)
	defer client.Close()
	ogBalance, err := getEthBalance(client, ethConfig.UsdcTokenAddress, destAddress)
	require.NoError(t, err)
	t.Logf("Destination Balance: %d", ogBalance)

	t.Logf("Burn Amount: %s", burnAmount.String())

	// set up sdk context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	sdkContext := sdkClient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}
	txBuilder := sdkContext.TxConfig.NewTxBuilder()

	// noble priveate key
	keyBz, _ := hex.DecodeString(nobleCfg.PrivateKey)
	privKey := secp256k1.PrivKey{Key: keyBz}
	nobleAddress, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.NoError(t, err)

	// destination address
	mintRecipient := make([]byte, 32)
	copy(mintRecipient[12:], common.FromHex(destAddress))

	// caller/relayer wallet address
	destinationCaller := make([]byte, 32)
	copy(destinationCaller[12:], common.FromHex(ethConfig.DestinationCaller[2:]))

	burnMsg := nobletypes.NewMsgDepositForBurnWithCaller(
		nobleAddress,
		burnAmount,
		ethConfig.Domain,
		mintRecipient,
		"uusdc",
		destinationCaller,
	)

	err = txBuilder.SetMsgs(burnMsg)
	require.NoError(t, err)

	txBuilder.SetGasLimit(200000)

	// sign + broadcast txn
	accountNumber, accountSequence, err := getNobleAccountNumberSequenceGRPC(cc, nobleAddress)
	require.NoError(t, err)

	sigV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: accountSequence,
	}

	signerData := xauthsigning.SignerData{
		ChainID:       nobleCfg.ChainID,
		AccountNumber: accountNumber,
		Sequence:      accountSequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	sigV2, err = clientTx.SignWithPrivKey(
		sdkContext.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		&privKey,
		sdkContext.TxConfig,
		accountSequence,
	)
	require.NoError(t, err)

	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	// Generated Protobuf-encoded bytes.
	txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	rpcResponse, err := cc.RPCClient.BroadcastTxSync(context.Background(), txBytes)
	require.NoError(t, err)
	t.Logf("Deposit for Burn broadcasted: https://mintscan.io/noble-testnet/txs/%s", rpcResponse.Hash.String())

	t.Log("Waiting for circle to approve and destination wallet to receive funds...")
	var newEthBalance uint64
	for i := 0; i < 120; i++ {
		newEthBalance, err = getEthBalance(client, ethConfig.UsdcTokenAddress, destAddress)

		require.NoError(t, err)
		if ogBalance+burnAmount.Uint64() == newEthBalance {
			t.Log("Successfully minted into " + destAddress)
			break
		}
		time.Sleep(3 * time.Second)
	}

	newNobleBal, err := getNobleAccountBalance(ctx, cc, nobleCfg.Address, uusdcDenom)
	require.NoError(t, err)
	t.Log("Source Balance: ", newNobleBal)

	// verify eth balance
	t.Logf("Destination Balance: %d", newEthBalance)
	require.Equal(t, ogBalance+burnAmount.Uint64(), newEthBalance)
}

// Test Eth/L2 burn to Noble deposit on actively deployed relayer
func TestEthBurnToNobleDeployed(t *testing.T) {
	c, err := ParseIntegration("../.ignore/integration.yaml")
	require.NoError(t, err)
	ctx := context.Background()

	// -- NETWORK --
	cfg := c.Testnet
	// cfg := c.Mainnet

	// -- SRC CHAIN --
	ethConfig := cfg["ethereum"]
	// ethConfig := cfg["arbitrum"]
	// ethConfig := cfg["avalanche"]
	// ethConfig := cfg["optimism"]
	// ethConfig := cfg["polygon"]
	// ethConfig := cfg["base"]

	// -- DEST CHAIN --
	nobleCfg := cfg["noble"]

	var burnAmount = big.NewInt(1)

	destAddress := nobleCfg.Address

	t.Logf("Path: %d -> %d", ethConfig.Domain, nobleCfg.Domain)

	t.Logf("Source Address: %s", ethConfig.Address)

	client, err := ethclient.Dial(ethConfig.RPC)
	require.NoError(t, err)
	defer client.Close()

	originalEthBalance, err := getEthBalance(client, ethConfig.UsdcTokenAddress, ethConfig.Address)
	require.NoError(t, err)
	t.Logf("Source Balance: %d", originalEthBalance)

	t.Logf("Destination Address: %s", destAddress)

	// Get original usdc balance on Noble to verify that funds are deposited later
	cc, err := cosmos.NewProvider(nobleCfg.RPC)
	require.NoError(t, err)
	originalNobleBalance, err := getNobleAccountBalance(ctx, cc, destAddress, uusdcDenom)
	require.NoError(t, err)
	t.Logf("Destination Balance: %d", originalNobleBalance)

	t.Logf("Burn Amount: %s", burnAmount.String())

	privateKey, err := crypto.HexToECDSA(ethConfig.PrivateKey)
	require.NoError(t, err)

	i, err := strconv.ParseInt(ethConfig.ChainID, 10, 64)
	require.NoError(t, err)
	ethChainID := big.NewInt(i)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, ethChainID)
	require.NoError(t, err)

	// deal w/ nonce
	nextNonce, err := ethereum.GetEthereumAccountNonce(ethConfig.RPC, ethConfig.Address)
	require.NoError(t, err)
	auth.Nonce = big.NewInt(nextNonce)

	// Approve erc20 to interact with contract up to the sum of the amount being burnt
	erc20, err := NewERC20(common.HexToAddress(ethConfig.UsdcTokenAddress), client)
	require.NoError(t, err)

	// contractValue := burnAmount
	_, err = erc20.Approve(auth, common.HexToAddress(ethConfig.TokenMessengerAddress), burnAmount)
	require.NoError(t, err)

	// wait for erc20 approval to be on chain
	time.Sleep(10 * time.Second)

	// create tokenMessenger
	tokenMessenger, err := contracts.NewTokenMessenger(common.HexToAddress(ethConfig.TokenMessengerAddress), client)
	require.NoError(t, err)

	_, mintRecipientRaw, err := bech32.DecodeAndConvert(nobleCfg.Address)
	require.NoError(t, err)
	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, mintRecipientRaw...)

	_, destinationCallerRaw, err := bech32.DecodeAndConvert(nobleCfg.DestinationCaller)
	require.NoError(t, err)
	destinationCallerPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, destinationCallerRaw...)

	auth.Nonce = big.NewInt(nextNonce + 1)

	tx, err := tokenMessenger.DepositForBurnWithCaller(
		auth,
		burnAmount,
		4,
		[32]byte(mintRecipientPadded),
		common.HexToAddress(ethConfig.UsdcTokenAddress),
		[32]byte(destinationCallerPadded),
	)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)
	t.Logf("Deposit for Burn broadcasted. Tx Hash: %s", tx.Hash().String())

	var newBalance uint64
	t.Log("Waiting for circle to approve and destination wallet to receive funds...")
	for i := 0; i < 250; i++ {
		newBalance, err = getNobleAccountBalance(ctx, cc, destAddress, uusdcDenom)
		require.NoError(t, err)
		if originalNobleBalance+burnAmount.Uint64() == newBalance {
			t.Logf("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + destAddress)
			break
		}
		time.Sleep(3 * time.Second)
	}

	newEthBalance, err := getEthBalance(client, ethConfig.UsdcTokenAddress, ethConfig.Address)
	require.NoError(t, err)
	t.Logf("Source Balance: %d", newEthBalance)

	t.Logf("Destination Balance: %d", newBalance)
	// verify noble balance
	require.Equal(t, originalNobleBalance+burnAmount.Uint64(), newBalance)
}

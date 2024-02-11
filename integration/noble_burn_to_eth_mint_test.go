package integration_testing

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

// TestNobleBurnToEthMint broadcasts a depositForBurn on Noble of 1 cent
// to a randomly generated Sepolia wallet
//
// Requires:
//   - Config is setup and config file is passed into cmd.Parse.
//   - Config must have a chain named "noble" (grand-1 testnet) and another named "sepolia"
//   - must have at least  1uusdc in Noble relayer wallet
//
// This test uses the Sepolia wallet in the config as the destination caller to ensure that
// to ensure that this instance of the relayer picks up the transaction
func TestNobleBurnToEthMint(t *testing.T) {
	ctx := context.Background()
	logger := log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))

	// Relayer config
	cfg, err := cmd.Parse("../.ignore/testnet.yaml")
	require.NoError(t, err)

	nobleCfg := cfg.Chains["noble"].(*noble.ChainConfig)
	ethCfg := cfg.Chains["sepolia"].(*ethereum.ChainConfig)

	nobleChain, err := nobleCfg.Chain("noble")
	require.NoError(t, err)
	ethChain, err := ethCfg.Chain("eth")
	require.NoError(t, err)

	fmt.Println("Starting relayer...")

	registeredDomains := make(map[types.Domain]types.Chain)
	registeredDomains[0] = ethChain
	registeredDomains[4] = nobleChain

	sequenceMap := types.NewSequenceMap()
	err = ethChain.InitializeBroadcaster(ctx, logger, sequenceMap)
	require.NoError(t, err)

	processingQueue := make(chan *types.TxState, 10)

	go nobleChain.StartListener(ctx, logger, processingQueue)
	go cmd.StartProcessor(ctx, cfg, logger, registeredDomains, processingQueue, sequenceMap)

	ethDestinationAddress, _, err := generateEthWallet()
	require.NoError(t, err)
	fmt.Println("Generated dest wallet: ", ethDestinationAddress)

	fmt.Println("Minting on Ethereum to https://sepolia.etherscan.io/address/" + ethDestinationAddress)

	// verify ethereum usdc amount
	client, _ := ethclient.Dial(ethCfg.RPC)
	defer client.Close()
	originalEthBalance, err := getEthBalance(client, usdcTokenAddressSepolia, ethDestinationAddress)
	require.NoError(t, err)

	// deposit for burn

	// set up sdk context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	sdkContext := sdkClient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}
	txBuilder := sdkContext.TxConfig.NewTxBuilder()
	// get priv key
	// keyBz, _ := hex.DecodeString(integrationWallets.Networks.Noble.PrivateKey)
	keyBz, _ := hex.DecodeString(nobleCfg.MinterPrivateKey)

	privKey := secp256k1.PrivKey{Key: keyBz}
	nobleAddress, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.Nil(t, err)

	mintRecipient := make([]byte, 32)
	copy(mintRecipient[12:], common.FromHex(ethDestinationAddress))
	var burnAmount = math.NewInt(1)

	// caller/relayer wallet address
	// need to convert private key from hex to public address
	callerPrivKey := ethCfg.MinterPrivateKey
	privateKeyBytes := common.FromHex(callerPrivKey)
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	require.Nil(t, err)
	pubKey := privateKey.Public()
	publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	caller := crypto.PubkeyToAddress(*publicKeyECDSA).String()

	destinationCaller := make([]byte, 32)
	copy(destinationCaller[12:], common.FromHex(caller[2:]))

	burnMsg := nobletypes.NewMsgDepositForBurnWithCaller(
		nobleAddress,
		burnAmount,
		uint32(0),
		mintRecipient,
		"uusdc",
		destinationCaller,
	)

	err = txBuilder.SetMsgs(burnMsg)
	require.Nil(t, err)

	txBuilder.SetGasLimit(nobleCfg.GasLimit)

	// sign + broadcast txn
	cc, err := cosmos.NewProvider(nobleCfg.RPC)
	require.Nil(t, err)

	accountNumber, accountSequence, err := getNobleAccountNumberSequenceGRPC(cc, nobleAddress)

	require.Nil(t, err)

	sigV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: uint64(accountSequence),
	}

	signerData := xauthsigning.SignerData{
		ChainID:       nobleCfg.ChainID,
		AccountNumber: uint64(accountNumber),
		Sequence:      uint64(accountSequence),
	}

	txBuilder.SetSignatures(sigV2)
	sigV2, err = clientTx.SignWithPrivKey(
		sdkContext.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		&privKey,
		sdkContext.TxConfig,
		uint64(accountSequence),
	)
	require.Nil(t, err)

	err = txBuilder.SetSignatures(sigV2)
	require.Nil(t, err)

	// Generated Protobuf-encoded bytes.
	txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.Nil(t, err)

	rpcResponse, err := cc.RPCClient.BroadcastTxSync(context.Background(), txBytes)
	require.Nil(t, err)
	fmt.Printf("Update pending: https://testnet.mintscan.io/noble-testnet/txs/%s\n", rpcResponse.Hash.String())

	fmt.Println("Checking eth wallet...")
	var newBalance uint64
	for i := 0; i < 60; i++ {
		newBalance, err = getEthBalance(client, usdcTokenAddressSepolia, ethDestinationAddress)
		require.NoError(t, err)
		if originalEthBalance+burnAmount.Uint64() == newBalance {
			fmt.Println("Successfully minted at https://sepolia.etherscan.io/address/" + ethDestinationAddress)
			break
		}
		time.Sleep(3 * time.Second)
	}
	// verify eth balance
	require.Equal(t, originalEthBalance+burnAmount.Uint64(), newBalance)
}

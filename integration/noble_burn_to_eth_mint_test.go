package integration_testing

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"

	"github.com/stretchr/testify/require"
)

// TestNobleBurnToEthMint generates and broadcasts a depositForBurn on Noble
// and broadcasts on Ethereum Goerli
func TestNobleBurnToEthMint(t *testing.T) {
	ctx := context.Background()

	setupTestIntegration()

	fmt.Println("Starting relayer...")
	processingQueue := make(chan *types.TxState, 10)

	registeredDomains := make(map[types.Domain]types.Chain)
	registeredDomains[0] = ethChain
	registeredDomains[4] = nobleChain

	err := ethChain.InitializeBroadcaster(ctx, logger, sequenceMap)
	require.NoError(t, err)

	go nobleChain.StartListener(ctx, logger, processingQueue)
	go cmd.StartProcessor(context.TODO(), cfg, logger, registeredDomains, processingQueue, sequenceMap)

	fmt.Println("Building Noble depositForBurn txn...")
	ethDestinationAddress := integrationWallets.Networks.Ethereum.Address
	fmt.Println("Minting on Ethereum to https://sepolia.etherscan.io/address/" + ethDestinationAddress)

	// verify ethereum usdc amount
	client, _ := ethclient.Dial(ethCfg.RPC)
	defer client.Close()
	originalEthBalance := getEthBalance(client, ethDestinationAddress)

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
	keyBz, _ := hex.DecodeString(integrationWallets.Networks.Noble.PrivateKey)
	privKey := secp256k1.PrivKey{Key: keyBz}
	nobleAddress, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.Nil(t, err)

	mintRecipient := make([]byte, 32)
	copy(mintRecipient[12:], common.FromHex(ethDestinationAddress))
	var burnAmount = math.NewInt(1)

	// deposit for burn on noble
	burnMsg := nobletypes.NewMsgDepositForBurn(
		nobleAddress,
		burnAmount,
		uint32(0),
		mintRecipient,
		"uusdc",
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
	for i := 0; i < 60; i++ {
		if originalEthBalance+burnAmount.Uint64() == getEthBalance(client, ethDestinationAddress) {
			fmt.Println("Successfully minted at https://sepolia.etherscan.io/address/" + ethDestinationAddress)
			return
		}
		time.Sleep(3 * time.Second)
	}
	// verify eth balance
	require.Equal(t, originalEthBalance+burnAmount.Uint64(), getEthBalance(client, ethDestinationAddress))
}

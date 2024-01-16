package integration_testing

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

// TestEthereumMultiSend broadcasts N depositForBurnWithCaller messages on Noble, and then tries to receive them all at once on Ethereum.
// We require a destination caller in this test so the deployed relayer doesn't pick it up.
//
// The point of this test is to verify that the Ethereum minter's account sequence is synced.
// A successful test means that all messages went through without retries (which are set to zero).
// We verify this result by checking the account balance at the end of the test.
func TestEthereumMultiSend(t *testing.T) {
	setupTest()

	// number of depositForBurn txns to send
	n := 10

	ethMultiSendCfg := Parse("../.ignore/eth_multi_send.yaml")

	cfg.Networks.Source.Noble.StartBlock = getNobleLatestBlockHeight()
	cfg.Networks.Destination.Ethereum.BroadcastRetries = 0 // don't rely on retries to broadcast txns

	// the caller account functions both as the destination caller and minter
	callerPrivKey := ethMultiSendCfg.Networks.Ethereum.PrivateKey
	privateKeyBytes := common.FromHex(callerPrivKey)
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	require.Nil(t, err)
	pubKey := privateKey.Public()
	publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	caller := crypto.PubkeyToAddress(*publicKeyECDSA).String()

	for i, minter := range cfg.Networks.Minters {
		switch i {
		case 0:
			minter.MinterAddress = caller
			minter.MinterPrivateKey = callerPrivKey
			cfg.Networks.Minters[0] = minter
		}
	}

	nonce, err := eth.GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, caller)
	require.Nil(t, err)

	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(0), nonce)

	fmt.Println(fmt.Sprintf("Building %d Noble depositForBurnWithCaller txns...", n))

	ethMintRecipient := "0x971c54a6Eb782fAccD00bc3Ed5E934Cc5bD8e3Ef"
	fmt.Println("Minting on Ethereum to https://goerli.etherscan.io/address/" + ethMintRecipient)

	// verify original eth usdc amount
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()
	originalEthBalance := getEthBalance(client, ethMintRecipient)
	fmt.Println(fmt.Sprintf("original usdc balance: %d", originalEthBalance))

	// deposit for burn with caller
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	sdkContext := sdkClient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}
	txBuilder := sdkContext.TxConfig.NewTxBuilder()

	// get priv key
	keyBz, _ := hex.DecodeString(testCfg.Networks.Noble.PrivateKey)
	privKey := secp256k1.PrivKey{Key: keyBz}
	nobleAddress, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.Nil(t, err)

	mintRecipient := make([]byte, 32)
	copy(mintRecipient[12:], common.FromHex(ethMintRecipient))

	destinationCaller := make([]byte, 32)
	copy(destinationCaller[12:], common.FromHex(caller[2:]))

	var burnAmount = math.NewInt(1)

	// deposit for burn on noble
	accountNumber, accountSequence, err := GetNobleAccountNumberSequence(cfg.Networks.Destination.Noble.API, nobleAddress)
	require.Nil(t, err)

	for i := 0; i < n; i++ {
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

		txBuilder.SetGasLimit(300000)

		// sign + broadcast txn
		rpcClient, err := NewRPCClient(testCfg.Networks.Noble.RPC, 10*time.Second)
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
			ChainID:       cfg.Networks.Destination.Noble.ChainId,
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

		err = txBuilder.SetSignatures(sigV2)
		require.Nil(t, err)

		// Generated Protobuf-encoded bytes.
		txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
		require.Nil(t, err)

		rpcResponse, err := rpcClient.BroadcastTxSync(context.Background(), txBytes)
		require.Nil(t, err)
		fmt.Printf("Update pending: https://testnet.mintscan.io/noble-testnet/txs/%s\n", rpcResponse.Hash.String())

		accountSequence++
	}

	fmt.Println("Waiting 60 seconds for attestations...")
	time.Sleep(60 * time.Second)

	fmt.Println("Starting relayer...")

	processingQueue := make(chan *types.MessageState, 100)

	go noble.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(context.TODO(), cfg, logger, processingQueue, sequenceMap)

	fmt.Println("Checking eth wallet...")
	for i := 0; i < 60; i++ {
		if originalEthBalance+burnAmount.Uint64()*uint64(n) == getEthBalance(client, ethMintRecipient) {
			fmt.Println(fmt.Sprintf("New eth balance: %d", getEthBalance(client, ethMintRecipient)))
			fmt.Println(fmt.Sprintf("Successfully minted %d times at https://goerli.etherscan.io/address/%s", n, ethMintRecipient))
			return
		}
		time.Sleep(1 * time.Second)
	}

	require.Equal(t, originalEthBalance+burnAmount.Uint64()*uint64(n), getEthBalance(client, ethMintRecipient))
}

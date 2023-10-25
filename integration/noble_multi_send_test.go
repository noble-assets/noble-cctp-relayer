package integration_testing

import (
	"encoding/hex"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

// TestNobleMultiSend broadcasts N depositForBurnWithCaller messages on Ethereum, and then tries to receive them all at once on Noble.
// We require a destination caller in this test so the deployed relayer doesn't pick it up.
//
// The point of this test is to verify that the Noble minter's account sequence is synced.
// A successful test means that all messages went through without retries (which are set to zero).
// We verify this result by checking the account balance at the end of the test.
func TestNobleMultiSend(t *testing.T) {
	setupTest()

	nobleMultiSendCfg := Parse("../.ignore/noble_multi_send.yaml")

	// the caller account functions both as the destination caller and minter
	var callerPrivKey = nobleMultiSendCfg.Networks.Noble.PrivateKey
	keyBz, err := hex.DecodeString(callerPrivKey)
	require.Nil(t, err)
	privKey := secp256k1.PrivKey{Key: keyBz}
	caller, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	require.Nil(t, err)

	for i, minter := range cfg.Networks.Minters {
		switch i {
		case 4:
			minter.MinterAddress = caller
			minter.MinterPrivateKey = callerPrivKey
			cfg.Networks.Minters[4] = minter
		}
	}

	_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
		cfg.Networks.Destination.Noble.API,
		cfg.Networks.Minters[4].MinterAddress)

	require.Nil(t, err)

	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(4), nextMinterSequence)

	// number of depositForBurn txns to send
	n := 7

	// start up relayer
	cfg.Networks.Source.Ethereum.StartBlock = getEthereumLatestBlockHeight(t)
	cfg.Networks.Source.Ethereum.LookbackPeriod = 5
	cfg.Networks.Destination.Noble.BroadcastRetries = 0 // don't rely on retries to broadcast txns

	fmt.Println(fmt.Sprintf("Building %d Ethereum depositForBurnWithMetadata txns...", n))

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

	_, callerRaw, _ := bech32.DecodeAndConvert(caller)
	destinationCallerPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, callerRaw...)
	require.Nil(t, err)

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerWithMetadataAddress), big.NewInt(99999))
	require.Nil(t, err)
	var burnAmount = big.NewInt(1)

	for i := 0; i < n; i++ {
		tx, err := tokenMessenger.DepositForBurnWithCaller(
			auth,
			burnAmount,
			4,
			[32]byte(mintRecipientPadded),
			common.HexToAddress(UsdcAddress),
			[32]byte(destinationCallerPadded),
		)
		if err != nil {
			logger.Error("Failed to update value: %v", err)
		}

		time.Sleep(1 * time.Second)

		fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

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
			fmt.Println(fmt.Sprintf("Successfully minted %d times at https://testnet.mintscan.io/noble-testnet/account/%s", n, nobleAddress))
			return
		}
		time.Sleep(1 * time.Second)
	}
}

package demo

import (
	"context"
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	eth "github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

var testCfg Config    // for testing secrets
var cfg config.Config // app config
var logger log.Logger

// goerli
const TokenMessengerWithMetadataAddress = "0x1ae045d99236365cbdc1855acd2d2cfc232d04d1"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"

type BalanceResponse struct {
	Balance Coin `json:"balance"`
}

type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

func setupTest() func() {
	// setup
	testCfg = Parse("../.ignore/integration.yaml")
	cfg = config.Parse("../.ignore/testnet.yaml")
	logger = log.NewLogger(os.Stdout)

	return func() {
		// teardown
	}
}

// TestGenerateEthDepositForBurn generates and broadcasts a depositForBurnWithMetadata on Ethereum Goerli
func TestGenerateEthDepositForBurnWithForward(t *testing.T) {
	setupTest()

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
	destinationRecipient := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, cosmosAddress...)

	burnAmount := big.NewInt(5)

	tx, err := tokenMessengerWithMetadata.DepositForBurn(
		auth,
		channel,                           // channel
		[32]byte(destinationBech32Prefix), // destinationBech32Prefix
		[32]byte(destinationRecipient),    // destinationRecipient
		burnAmount,                        // amount
		[32]byte(mintRecipientPadded),     // mint recipient
		common.HexToAddress(UsdcAddress),  // burn token
		[]byte{},                          // memo
	)
	if err != nil {
		logger.Error("Failed to update value: %v", err)
	}

	time.Sleep(5 * time.Second)
	fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

	fmt.Println("Waiting 100 seconds for the attestation to finalize...")
	time.Sleep(100 * time.Second)

	// start up relayer
	// set height to current block height
	cfg.Networks.Source.Ethereum.StartBlock = getEthereumLatestBlockHeight(t)
	cfg.Networks.Source.Ethereum.LookbackPeriod = 20

	fmt.Println("Relaying...")
	processingQueue := make(chan *types.MessageState, 10)
	go eth.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(cfg, logger, processingQueue)

	time.Sleep(30 * time.Second)
	// verify Dydx balance
	require.Equal(t, originalDydx+burnAmount.Uint64(), getDydxBalance(dydxAddress))

	fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)
}

func getDydxBalance(address string) uint64 {
	rawResponse, _ := http.Get(fmt.Sprintf(
		"https://dydx-testnet-api.polkachu.com/cosmos/bank/v1beta1/balances/%s/by_denom?denom=ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5", address))
	body, _ := io.ReadAll(rawResponse.Body)
	response := BalanceResponse{}
	_ = json.Unmarshal(body, &response)
	res, _ := strconv.ParseInt(response.Balance.Amount, 0, 0)
	return uint64(res)
}

func getEthereumLatestBlockHeight(t *testing.T) uint64 {
	client, err := ethclient.Dial(cfg.Networks.Source.Ethereum.RPC)
	require.Nil(t, err)

	header, err := client.HeaderByNumber(context.Background(), nil)
	require.Nil(t, err)
	return header.Number.Uint64()
}

package integration_testing

import (
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
	"testing"
	"time"
)

var testCfg Config    // for testing secrets
var cfg config.Config // app config
var logger log.Logger

// goerli
const TokenMessengerAddress = "0xd0c3da58f55358142b8d3e06c1c30c5c6114efe8"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"

func setupTest() func() {
	// setup
	testCfg = Parse("../.ignore/integration.yaml")
	cfg = config.Parse("../.ignore/testnet.yaml")
	logger = log.NewLogger(os.Stdout)

	return func() {
		// teardown
	}
}

type Coin struct {
	Denom  string
	Amount big.Int
}

// TestGenerateEthDepositForBurn generates and broadcasts a depositForBurn on Ethereum Goerli
func TestGenerateEthDepositForBurn(t *testing.T) {
	setupTest()

	processingQueue := make(chan *types.MessageState, 10)
	go eth.StartListener(cfg, logger, processingQueue)
	go cmd.StartProcessor(cfg, logger, processingQueue)

	_, _, cosmosAddress := testdata.KeyTestPubAddr()
	nobleAddress, _ := bech32.ConvertAndEncode("noble", cosmosAddress.Bytes())

	fmt.Print("Minting to " + nobleAddress)

	// verify noble address usdc amount
	nobleAddressBalance := getBalance(nobleAddress)

	// deposit for burn
	client, err := ethclient.Dial(testCfg.Networks.Ethereum.RPC)
	require.Nil(t, err)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(testCfg.Networks.Ethereum.PrivateKey)
	require.Nil(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))
	require.Nil(t, err)
	// set nonce
	//nextNonce, err := client.PendingNonceAt(context.Background(), auth.From)
	//require.Nil(t, err)
	//auth.Nonce = big.NewInt(int64(nextNonce))

	tokenMessenger, err := cmd.NewTokenMessenger(common.HexToAddress(TokenMessengerAddress), client)
	require.Nil(t, err)

	_, mintRecipientBz, _ := bech32.DecodeAndConvert(nobleAddress)
	mintRecipientPadded := append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, mintRecipientBz...)
	require.Nil(t, err)

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddress), big.NewInt(99999))
	require.Nil(t, err)

	burnAmount := big.NewInt(1)
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
	fmt.Printf("Update pending: https://goerli.etherscan.io/tx/0x%x\n", tx.Hash())

	fmt.Println("Waiting 90 seconds for the attestation to finalize...")
	time.Sleep(90 * time.Second)

	// verify burned USDC has showed up in Noble
	require.Equal(t, nobleAddressBalance+burnAmount.Uint64(), getBalance(nobleAddress))

	fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)
}

func getBalance(address string) uint64 {
	rawResponse, _ := http.Get(fmt.Sprintf("https://api.testnet.noble.strange.love/cosmos/bank/v1beta1/balances/%s/by_denom?denom=uusdc", address))
	body, _ := io.ReadAll(rawResponse.Body)
	response := Coin{}
	_ = json.Unmarshal(body, &response)
	return response.Amount.Uint64()
}
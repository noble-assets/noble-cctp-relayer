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
	nobleAddress, _ := bech32.ConvertAndEncode("noble", cosmosAddress)

	fmt.Println("Minting to " + nobleAddress)

	// verify noble address usdc amount
	nobleAddressBalance := getNobleBalance(nobleAddress)

	// deposit for burn
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

	erc20, err := NewERC20(common.HexToAddress(UsdcAddress), client)
	_, err = erc20.Approve(auth, common.HexToAddress(TokenMessengerAddress), big.NewInt(99999))
	require.Nil(t, err)

	// flakey
	burnAmount := big.NewInt(163)

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

	time.Sleep(2 * time.Second)
	fmt.Printf("Update pending: https://goerli.etherscan.io/tx/%s\n", tx.Hash().String())

	fmt.Println("Waiting 90 seconds for the attestation to finalize...")
	time.Sleep(90 * time.Second)

	// verify Noble balance
	require.Equal(t, nobleAddressBalance+burnAmount.Uint64(), getNobleBalance(nobleAddress))

	fmt.Println("Successfully minted at https://testnet.mintscan.io/noble-testnet/account/" + nobleAddress)
}

func getNobleBalance(address string) uint64 {
	rawResponse, _ := http.Get(fmt.Sprintf("https://api.testnet.noble.strange.love/cosmos/bank/v1beta1/balances/%s/by_denom?denom=uusdc", address))
	body, _ := io.ReadAll(rawResponse.Body)
	response := Coin{}
	_ = json.Unmarshal(body, &response)
	return response.Amount.Uint64()
}

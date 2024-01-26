package integration_testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/stretchr/testify/require"
)

// USDC Token Address on Sepolia
const usdcTokenAddress = "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"

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
	client, err := ethclient.Dial(ethCfg.RPC)
	require.Nil(t, err)

	header, err := client.HeaderByNumber(context.Background(), nil)
	require.Nil(t, err)
	return header.Number.Uint64()
}

func getNobleLatestBlockHeight() uint64 {
	rawResponse, _ := http.Get("https://rpc.testnet.noble.strange.love/block")
	body, _ := io.ReadAll(rawResponse.Body)
	response := NobleBlockResponse{}
	_ = json.Unmarshal(body, &response)
	res, _ := strconv.ParseInt(response.Result.Block.Header.Height, 0, 0)
	return uint64(res)
}

func getNobleAccountBalance(ctx context.Context, cc *cosmos.CosmosProvider, address, denom string) (uint64, error) {
	qc := bankTypes.NewQueryClient(cc)
	res, err := qc.Balance(ctx, &bankTypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return 0, err
	}

	return res.Balance.Amount.Uint64(), nil
}

func getNobleAccountNumberSequenceGRPC(cc *cosmos.CosmosProvider, address string) (uint64, uint64, error) {
	res, err := authtypes.NewQueryClient(cc).Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query account for noble: %w", err)
	}
	var acc authtypes.AccountI
	if err := cc.Cdc.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, 0, fmt.Errorf("unable to unpack account for noble: %w", err)
	}

	return acc.GetAccountNumber(), acc.GetSequence(), nil

}

func getEthBalance(client *ethclient.Client, address string) uint64 {
	accountAddress := common.HexToAddress(address)
	tokenAddress := common.HexToAddress(usdcTokenAddress)
	erc20ABI := `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}

	data, err := parsedABI.Pack("balanceOf", accountAddress)
	if err != nil {
		log.Fatalf("Failed to pack data into ABI interface: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{To: &tokenAddress, Data: data}, nil)
	if err != nil {
		log.Fatalf("Failed to call contract: %v", err)
	}

	balance := new(big.Int)
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		log.Fatalf("Failed to unpack data from ABI interface: %v", err)
	}

	// Convert to uint64
	return balance.Uint64()
}

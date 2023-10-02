package integration_testing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strconv"
	"testing"
)

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

func getNobleBalance(address string) uint64 {
	rawResponse, _ := http.Get(fmt.Sprintf("https://lcd.testnet.noble.strange.love/cosmos/bank/v1beta1/balances/%s/by_denom?denom=uusdc", address))
	body, _ := io.ReadAll(rawResponse.Body)
	response := BalanceResponse{}
	_ = json.Unmarshal(body, &response)
	result, _ := strconv.ParseInt(response.Balance.Amount, 10, 0)
	return uint64(result)
}

func getNobleLatestBlockHeight() uint64 {
	rawResponse, _ := http.Get("https://rpc.testnet.noble.strange.love/block")
	body, _ := io.ReadAll(rawResponse.Body)
	response := NobleBlockResponse{}
	_ = json.Unmarshal(body, &response)
	res, _ := strconv.ParseInt(response.Result.Block.Header.Height, 0, 0)
	return uint64(res)
}

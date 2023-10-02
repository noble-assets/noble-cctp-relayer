package integration_testing

type BalanceResponse struct {
	Balance struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"balance"`
}

type NobleBlockResponse struct {
	Result struct {
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}

type EthereumRPCPayload struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type EthereumTxCountResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

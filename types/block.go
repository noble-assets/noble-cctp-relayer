package types

type BlockResponse struct {
	Result struct {
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}

type BlockResultsResponse struct {
	Result struct {
		Txs []Tx `json:"txs"`
	} `json:"result"`
}

type Tx struct {
	Hash     string `json:"hash"`
	TxResult struct {
		Code int    `json:"code"`
		Log  string `json:"log"`
	} `json:"tx_result"`
}

type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// BlockHeight defines the expected response structure when using Solana's
// getBlockHeight RPC method with the "finalized" commitment type.
type BlockHeight struct {
	Result uint64 `json:"result"`
}

// Transaction defines the expected response structure when using Solana's
// getTransaction RPC method with the "jsonParsed" encoding type.
type Transaction struct {
	Result struct {
		Meta struct {
			InnerInstructions []struct {
				Index        int `json:"index"`
				Instructions []struct {
					Accounts []string `json:"accounts"`
					Data     string   `json:"data"`
				} `json:"instructions"`
			} `json:"innerInstructions"`
			LogMessages []string `json:"logMessages"`
		} `json:"meta"`
		Transaction struct {
			Message struct {
				AccountKeys []struct {
					PubKey   string `json:"pubKey"`
					Signer   bool   `json:"signer"`
					Writable bool   `json:"writable"`
				} `json:"accountKeys"`
				Instructions []struct {
					Accounts []string `json:"accounts"`
					Data     string   `json:"data"`
				} `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
	} `json:"result"`
}

// GetBlockHeight is a utility that returns the response of a request using
// Solana's getBlockHeight RPC method with the "finalized" commitment type.
// TODO: Actually filter by "finalized" commitment type!
func (s *Solana) GetBlockHeight(ctx context.Context) (uint64, error) {
	data := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "getBlockHeight",
		"id": "%s"
	}`, uuid.New().String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoints.RPC, strings.NewReader(data))
	if err != nil {
		return 0, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var blockHeight BlockHeight
	if err = json.Unmarshal(body, &blockHeight); err != nil {
		return 0, err
	}

	return blockHeight.Result, nil
}

// GetTransaction is a utility that returns the response of a request using
// Solana's getTransaction RPC method with the "jsonParsed" encoding type.
func (s *Solana) GetTransaction(ctx context.Context, endpoint string, hash string) (*Transaction, error) {
	data := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "getTransaction",
		"id": "%s",
		"params": [
			"%s",
			{
				"encoding": "jsonParsed",
				"commitment": "finalized",
				"maxSupportedTransactionVersion": 0
			}
		]
	}`, uuid.New().String(), hash)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var tx Transaction
	if err = json.Unmarshal(body, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

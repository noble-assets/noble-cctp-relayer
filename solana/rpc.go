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

// AccountInfo defines the expected response structure when using Solana's
// getAccountInfo RPC method with the "base64" encoding type.
type AccountInfo struct {
	Result struct {
		Value struct {
			Data []string `json:"data"`
		} `json:"value"`
	} `json:"result"`
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

// GetAccountInfo is a utility that returns the response of a request using
// Solana's getAccountInfo RPC method with the "base64" encoding type.
func (s *Solana) GetAccountInfo(ctx context.Context, endpoint string, account string) (*AccountInfo, error) {
	data := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "getAccountInfo",
		"id": "%s",
		"params": [
			"%s",
			{ "encoding": "base64" }
		]
	}`, uuid.New().String(), account)

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

	var info AccountInfo
	if err = json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
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
				"commitment": "confirmed",
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

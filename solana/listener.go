package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/cosmos/btcutil/base58"

	"github.com/strangelove-ventures/noble-cctp-relayer/solana/generated/message_transmitter"
)

// Transaction defines the expected response structure when using Solana's
// getTransaction RPC method with the "jsonParsed" encoding type.
type Transaction struct {
	Result struct {
		Meta struct {
			InnerInstructions []struct {
				Index        int `json:"index"`
				Instructions []struct {
					Data string `json:"data"`
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
					Data string `json:"data"`
				} `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
	} `json:"result"`
}

// NewTransactionRequest is a utility that forms a request for using Solana's
// getTransaction RPC method with the "jsonParsed" encoding type.
func (s *Solana) NewTransactionRequest(endpoint string, hash string) (*http.Request, error) {
	data := fmt.Sprintf(`{"method":"getTransaction","jsonrpc":"2.0","params":["%s",{"encoding":"jsonParsed","commitment":"confirmed","maxSupportedTransactionVersion":0}],"id":"%s"}`, hash, uuid.New().String())

	// TODO: Receive the context from the main listener!
	return http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader(data))
}

// ParseTransaction is a utility that fetches a transaction from the Solana RPC
// and returns all messages sent via CCTP. It does not apply any filtering.
func (s *Solana) ParseTransaction(endpoint string, hash string) (events []message_transmitter.SendMessageWithCallerParams, err error) {
	req, err := s.NewTransactionRequest(endpoint, hash)
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

	var accounts []*solana.AccountMeta
	for _, account := range tx.Result.Transaction.Message.AccountKeys {
		accounts = append(accounts, &solana.AccountMeta{
			PublicKey:  solana.MustPublicKeyFromBase58(account.PubKey),
			IsWritable: account.Writable,
			IsSigner:   account.Signer,
		})
	}

	var instructions [][]byte
	for _, tmp := range tx.Result.Meta.InnerInstructions {
		for _, instruction := range tmp.Instructions {
			if instruction.Data != "" {
				instructions = append(instructions, base58.Decode(instruction.Data))
			}
		}
	}
	for _, instruction := range tx.Result.Transaction.Message.Instructions {
		if instruction.Data != "" {
			instructions = append(instructions, base58.Decode(instruction.Data))
		}
	}

	for _, bz := range instructions {
		rawEvent, err := message_transmitter.DecodeInstruction(accounts, bz)
		if err == nil {
			switch message_transmitter.InstructionIDToName(rawEvent.TypeID) {
			case "SendMessage":
				event := rawEvent.Impl.(*message_transmitter.SendMessage)
				events = append(events, message_transmitter.SendMessageWithCallerParams{
					DestinationDomain: event.Params.DestinationDomain,
					Recipient:         event.Params.Recipient,
					MessageBody:       event.Params.MessageBody,
					DestinationCaller: solana.PublicKey{},
				})
			case "SendMessageWithCaller":
				event := rawEvent.Impl.(*message_transmitter.SendMessageWithCaller)
				events = append(events, *event.Params)
			}
		}
	}

	return
}

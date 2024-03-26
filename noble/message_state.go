package noble

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// NobleLogToMessageState transforms a Noble log into a messageState
func txToMessageState(tx *ctypes.ResultTx) ([]*types.MessageState, error) {
	if tx.TxResult.Code != 0 {
		return nil, nil
	}

	var messageStates []*types.MessageState

	for _, event := range tx.TxResult.Events {
		if event.Type == "circle.cctp.v1.MessageSent" {
			//fmt.Printf("Saw cctp message %s - %d:%d\n", tx., i, j)
			var parsed bool
			var parseErrs error
			for _, attr := range event.Attributes {
				decodedKey, err := base64.StdEncoding.DecodeString(attr.Key)
				if err != nil {
					parseErrs = errors.Join(parseErrs, fmt.Errorf("failed to decode attribute key: %w", err))
				}
				if string(decodedKey) == "message" {
					// fmt.Printf("Saw message attribute %s - %d\n", tx.Hash, i)
					decodedValue, err := base64.StdEncoding.DecodeString(attr.Value)
					if err != nil {
						parseErrs = errors.Join(parseErrs, fmt.Errorf("error decoding attr.value: %w", err))
						continue
					}
					encoded := decodedValue[1 : len(decodedValue)-1]
					// Because we are using cometBFT v0.38, we need to decode the value twice.
					rawMessageSentBytes, err := base64.StdEncoding.DecodeString(string(encoded))
					if err != nil {
						parseErrs = errors.Join(parseErrs, fmt.Errorf("failed to decode message: %w", err))
						continue
					}

					hashed := crypto.Keccak256(rawMessageSentBytes)
					hashedHexStr := hex.EncodeToString(hashed)

					msg, err := new(types.Message).Parse(rawMessageSentBytes)
					if err != nil {
						parseErrs = errors.Join(parseErrs, fmt.Errorf("failed to parse message: %w", err))
						continue
					}

					parsed = true

					now := time.Now()

					messageState := &types.MessageState{
						IrisLookupId:      hashedHexStr,
						Status:            types.Created,
						SourceDomain:      types.Domain(msg.SourceDomain),
						DestDomain:        types.Domain(msg.DestinationDomain),
						Nonce:             msg.Nonce,
						SourceTxHash:      tx.Hash.String(),
						MsgSentBytes:      rawMessageSentBytes,
						MsgBody:           msg.MessageBody,
						DestinationCaller: msg.DestinationCaller,
						Created:           now,
						Updated:           now,
					}

					messageStates = append(messageStates, messageState)
				}
			}
			if !parsed {
				return nil, fmt.Errorf("unable to parse cctp message.  tx hash %s: %w", tx.Hash, parseErrs)
			}
		}
	}

	return messageStates, nil

}

package types

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	Created  string = "created"
	Pending  string = "pending"
	Attested string = "attested"
	Complete string = "complete"
	Failed   string = "failed"
	Filtered string = "filtered"

	Mint    string = "mint"
	Forward string = "forward"
)

type MessageState struct {
	IrisLookupId      string // hex encoded MessageSent bytes
	Type              string // 'mint' or 'forward'
	Status            string // created, pending, attested, complete, failed, filtered
	Attestation       string // hex encoded attestation
	SourceDomain      uint32 // source domain id
	DestDomain        uint32 // destination domain id
	SourceTxHash      string
	DestTxHash        string
	MsgSentBytes      []byte // bytes of the MessageSent message transmitter event
	DestinationCaller []byte // address authorized to call transaction
	Channel           string // "channel-%d" if a forward, empty if not a forward
	Created           time.Time
	Updated           time.Time
	Nonce             uint64
}

// EvmLogToMessageState transforms an evm log into a messageState given an ABI
func EvmLogToMessageState(abi abi.ABI, messageSent abi.Event, log *ethtypes.Log) (messageState *MessageState, err error) {
	event := make(map[string]interface{})
	_ = abi.UnpackIntoMap(event, messageSent.Name, log.Data)

	rawMessageSentBytes := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessageSentBytes)

	hashed := crypto.Keccak256(rawMessageSentBytes)
	hashedHexStr := hex.EncodeToString(hashed)

	messageState = &MessageState{
		IrisLookupId:      hashedHexStr,
		Status:            Created,
		SourceDomain:      message.SourceDomain,
		DestDomain:        message.DestinationDomain,
		SourceTxHash:      log.TxHash.Hex(),
		MsgSentBytes:      rawMessageSentBytes,
		DestinationCaller: message.DestinationCaller,
		Nonce:             message.Nonce,
		Created:           time.Now(),
		Updated:           time.Now(),
	}

	if _, err := new(BurnMessage).Parse(message.MessageBody); err == nil {
		messageState.Type = Mint
		return messageState, nil
	}

	if forward, err := new(MetadataMessage).Parse(message.MessageBody); err == nil {
		messageState.Type = Forward
		// add forward channel to object so we can filter later
		messageState.Channel = "channel-" + strconv.Itoa(int(forward.Channel))
		return messageState, nil
	}

	return nil, fmt.Errorf("unable to parse tx into message, tx hash %s", log.TxHash.Hex())
}

// NobleLogToMessageState transforms a Noble log into a messageState
func NobleLogToMessageState(tx Tx) ([]*MessageState, error) {
	var eventsList []struct {
		Events []Event `json:"events"`
	}
	if tx.TxResult.Log == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(tx.TxResult.Log), &eventsList); err != nil {
		return nil, errors.New("unable to parse log events")
	}

	var messageStates []*MessageState

	for i, log := range eventsList {
		for j, event := range log.Events {
			if event.Type == "circle.cctp.v1.MessageSent" {
				fmt.Printf("Saw cctp message %s - %d:%d\n", tx.Hash, i, j)
				var parsed bool
				var parseErrs error
				for _, attr := range event.Attributes {
					if attr.Key == "message" {
						fmt.Printf("Saw message attribute %s - %d:%d\n", tx.Hash, i, j)
						encoded := attr.Value[1 : len(attr.Value)-1]
						rawMessageSentBytes, err := base64.StdEncoding.DecodeString(encoded)
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

						messageState := &MessageState{
							IrisLookupId:      hashedHexStr,
							Type:              Mint,
							Status:            Created,
							SourceDomain:      msg.SourceDomain,
							DestDomain:        msg.DestinationDomain,
							SourceTxHash:      tx.Hash,
							MsgSentBytes:      rawMessageSentBytes,
							DestinationCaller: msg.DestinationCaller,
							Created:           time.Now(),
							Updated:           time.Now(),
						}

						messageStates = append(messageStates, messageState)

						fmt.Printf("Appended transfer from 4 to %d\n", msg.DestinationDomain)
					}
				}
				if !parsed {
					return nil, fmt.Errorf("unable to parse cctp message.  tx hash %s: %w", tx.Hash, parseErrs)
				}
			}
		}
	}

	return messageStates, nil

}

// DecodeDestinationCaller transforms an encoded Noble cctp address into a noble bech32 address
// left padded input -> bech32 output
func DecodeDestinationCaller(input []byte) (string, error) {
	if len(input) <= 12 {
		return "", errors.New("destinationCaller is too short")
	}
	output, err := bech32.ConvertAndEncode("noble", input[12:])
	if err != nil {
		return "", errors.New("unable to encode destination caller")
	}
	return output, nil
}

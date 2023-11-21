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
		Type:              Mint,
		Status:            Created,
		SourceDomain:      message.SourceDomain,
		DestDomain:        message.DestinationDomain,
		SourceTxHash:      log.TxHash.Hex(),
		MsgSentBytes:      rawMessageSentBytes,
		DestinationCaller: message.DestinationCaller,
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

	return nil, errors.New(fmt.Sprintf("unable to parse tx into message, tx hash %s", log.TxHash.Hex()))
}

// NobleLogToMessageState transforms a Noble log into a messageState
func NobleLogToMessageState(tx Tx) (messageState *MessageState, err error) {

	var eventsList []struct {
		Events []Event `json:"events"`
	}
	err = json.Unmarshal([]byte(tx.TxResult.Log), &eventsList)
	if err != nil {
		return nil, errors.New("unable to parse log events")
	}

	for _, event := range eventsList[0].Events {
		if event.Type == "circle.cctp.v1.MessageSent" {
			for _, attr := range event.Attributes {
				if attr.Key == "message" {
					encoded := attr.Value[1 : len(attr.Value)-1]
					rawMessageSentBytes, err := base64.StdEncoding.DecodeString(encoded)
					if err != nil {
						continue
					}

					hashed := crypto.Keccak256(rawMessageSentBytes)
					hashedHexStr := hex.EncodeToString(hashed)

					msg, err := new(types.Message).Parse(rawMessageSentBytes)
					if err != nil {
						continue
					}

					messageState = &MessageState{
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

					return messageState, nil
				}
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("unable to parse txn into message.  tx hash %s", tx.Hash))
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

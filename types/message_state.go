package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strconv"
	"time"
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
	Attestation       string // hex encoded attestation, prepended with "0x"
	SourceDomain      uint32 // source domain id
	DestDomain        uint32 // destination domain id
	SourceTxHash      string
	DestTxHash        string
	MsgSentBytes      []byte
	DestinationCaller []byte // address authorized to call transaction
	Channel           string // "channel-%d" if a forward, empty if not a forward
	Created           time.Time
	Updated           time.Time
}

func ToMessageState(abi abi.ABI, messageSent abi.Event, log *ethtypes.Log) (messageState *MessageState, err error) {

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

	return nil, errors.New(fmt.Sprintf("unable to parse txn into message.  tx hash %s", log.TxHash.Hex()))
}

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

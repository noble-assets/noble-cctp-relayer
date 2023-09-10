package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"time"
)

const (
	Created  string = "created"
	Pending  string = "pending"
	Attested string = "attested"
	Complete string = "complete"
	Failed   string = "failed"

	Mint    string = "mint"
	Forward string = "forward"
)

type MessageState struct {
	IrisLookupId      string
	Type              string
	Status            string
	Attestation       string // hex encoded
	SourceDomain      uint32
	DestDomain        uint32
	SourceTxHash      string
	DestTxHash        string
	MsgSentBytes      []byte
	DestinationCaller []byte
	Created           time.Time
	Updated           time.Time
}

func ToMessageState(config config.Config, abi abi.ABI, messageSent abi.Event, log *ethtypes.Log) (messageState *MessageState, err error) {

	event := make(map[string]interface{})
	_ = abi.UnpackIntoMap(event, messageSent.Name, log.Data)

	rawMessageSentBytes := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessageSentBytes)

	if message.DestinationDomain != config.Networks.Noble.DomainId {
		return nil, errors.New("received message not intended for Noble")
	}

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

	// TODO add checks for if there's a destination caller (ignored state)
	if burn, err := new(BurnMessage).Parse(message.MessageBody); err == nil {
		cmd.Logger.Info("received a new burn", "nonce", message.Nonce, "tx", log.TxHash)
		messageState.Type = Mint
		return messageState, nil
	}

	if forward, err := new(MetadataMessage).Parse(message.MessageBody); err == nil {
		cmd.Logger.Info("received a new forward", "channel", forward.Channel, "tx", log.TxHash)
		messageState.Type = Forward
		return messageState, nil
	}

	return nil, errors.New(fmt.Sprintf("unable to parse txn into message.  tx hash %s", log.TxHash.Hex()))
}

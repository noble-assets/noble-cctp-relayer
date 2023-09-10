package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"os"
	"time"
)

const (
	Created  string = "created"
	Pending  string = "pending"
	Attested string = "attested"
	Complete string = "complete"

	Mint    string = "mint"
	Forward string = "forward"
)

type MessageState struct {
	IrisLookupId string
	Type         string
	Status       string
	Attestation  string // hex encoded
	SourceDomain uint32
	DestDomain   uint32
	SourceTxHash string
	DestTxHash   string
	MsgSentBytes []byte
	Created      time.Time
	Updated      time.Time
}

func ToMessageState(config config.Config, log *ethtypes.Log) (messageState *MessageState, err error) {
	messageTransmitter, err := os.Open("./abi/MessageTransmitter.json")
	if err != nil {
		//logger.Error("unable to read MessageTransmitter abi", "err", err)
		os.Exit(1)
	}
	MessageTransmitterABI, err := abi.JSON(messageTransmitter)
	if err != nil {
		//logger.Error("unable to parse MessageTransmitter abi", "err", err)
	}

	MessageSent := MessageTransmitterABI.Events["MessageSent"]

	event := make(map[string]interface{})
	_ = MessageTransmitterABI.UnpackIntoMap(event, MessageSent.Name, log.Data)

	rawMessageSentBytes := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessageSentBytes)

	if message.DestinationDomain != config.Networks.Noble.DomainId {
		return nil, errors.New("received message not intended for Noble")
	}

	// hexRaw, _ := hex.DecodeString("000000000000000000000004000000000003950D000000000000000000000000D0C3DA58F55358142B8D3E06C1C30C5C6114EFE800000000000000000000000057D4EAF1091577A6B7D121202AFBD2808134F11700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007865C6E87B9F70255377E024ACE6630C1EAA37F000000000000000000000000580B5AFD4B41B887339EA92D98F88BF07AAF04F500000000000000000000000000000000000000000000000000000000000F4240000000000000000000000000DB86162D6E6B273A95BD4D20FAADB83D7B5FE1CA")
	hashed := crypto.Keccak256(rawMessageSentBytes)
	hashedHexStr := hex.EncodeToString(hashed)

	messageState = &MessageState{
		IrisLookupId: hashedHexStr,
		Type:         Mint,
		Status:       Created,
		SourceDomain: message.SourceDomain,
		DestDomain:   message.DestinationDomain,
		SourceTxHash: log.TxHash.Hex(),
		MsgSentBytes: rawMessageSentBytes,
		Created:      time.Now(),
		Updated:      time.Now(),
	}

	if _, err := new(BurnMessage).Parse(message.MessageBody); err == nil {
		// logger.Info("received a new burn message", "nonce", message.Nonce, "tx", log.TxHash)
		messageState.Type = Mint
		return messageState, nil
	}

	if _, err := new(MetadataMessage).Parse(message.MessageBody); err == nil {
		// logger.Info("received a new forward message", "channel", content.Channel, "tx", log.TxHash)
		messageState.Type = Forward
		return messageState, nil
	}

	return nil, errors.New(fmt.Sprintf("unable to parse txn into message.  tx hash %s", log.TxHash.Hex()))
}

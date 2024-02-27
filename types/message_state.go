package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/circlefin/noble-cctp/x/cctp/types"
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

type Domain uint32

type TxState struct {
	TxHash string
	Msgs   []*MessageState
}

type MessageState struct {
	IrisLookupId      string // hex encoded MessageSent bytes
	Status            string // created, pending, attested, complete, failed, filtered
	Attestation       string // hex encoded attestation
	SourceDomain      Domain // uint32 source domain id
	DestDomain        Domain // uint32 destination domain id
	SourceTxHash      string
	DestTxHash        string
	MsgSentBytes      []byte // bytes of the MessageSent message transmitter event
	MsgBody           []byte // bytes of the MessageBody
	DestinationCaller []byte // address authorized to call transaction
	Channel           string // "channel-%d" if a forward, empty if not a forward
	Created           time.Time
	Updated           time.Time
	Nonce             uint64
}

// EvmLogToMessageState transforms an evm log into a messageState given an ABI
func EvmLogToMessageState(abi abi.ABI, messageSent abi.Event, log *ethtypes.Log) (messageState *MessageState, err error) {
	event := make(map[string]interface{})
	if err = abi.UnpackIntoMap(event, messageSent.Name, log.Data); err != nil {
		return nil, fmt.Errorf("unable to unpack evm log. error: %w", err)
	}

	rawMessageSentBytes := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessageSentBytes)

	hashed := crypto.Keccak256(rawMessageSentBytes)
	hashedHexStr := hex.EncodeToString(hashed)

	messageState = &MessageState{
		IrisLookupId:      hashedHexStr,
		Status:            Created,
		SourceDomain:      Domain(message.SourceDomain),
		DestDomain:        Domain(message.DestinationDomain),
		SourceTxHash:      log.TxHash.Hex(),
		MsgSentBytes:      rawMessageSentBytes,
		MsgBody:           message.MessageBody,
		DestinationCaller: message.DestinationCaller,
		Nonce:             message.Nonce,
		Created:           time.Now(),
		Updated:           time.Now(),
	}

	if _, err := new(BurnMessage).Parse(message.MessageBody); err == nil {
		return messageState, nil
	}

	return nil, fmt.Errorf("unable to parse tx into message, tx hash %s", log.TxHash.Hex())
}

// Equal checks if two MessageState instances are equal
func (m *MessageState) Equal(other *MessageState) bool {
	return (m.IrisLookupId == other.IrisLookupId &&
		m.Status == other.Status &&
		m.Attestation == other.Attestation &&
		m.SourceDomain == other.SourceDomain &&
		m.DestDomain == other.DestDomain &&
		m.SourceTxHash == other.SourceTxHash &&
		m.DestTxHash == other.DestTxHash &&
		bytes.Equal(m.MsgSentBytes, other.MsgSentBytes) &&
		bytes.Equal(m.DestinationCaller, other.DestinationCaller) &&
		m.Channel == other.Channel &&
		m.Created == other.Created &&
		m.Updated == other.Updated)
}

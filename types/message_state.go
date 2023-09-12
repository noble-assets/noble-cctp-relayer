package types

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
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
	Filtered string = "filtered"

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
		cmd.Logger.Info("received a new burn", "nonce", message.Nonce, "tx", log.TxHash)
		messageState.Type = Mint
		return messageState, nil
	}

	if forward, err := new(MetadataMessage).Parse(message.MessageBody); err == nil {
		cmd.Logger.Info("received a new forward", "channel", forward.Channel, "tx", log.TxHash)
		messageState.Type = Forward
		// TODO forward.
		return messageState, nil
	}

	return nil, errors.New(fmt.Sprintf("unable to parse txn into message.  tx hash %s", log.TxHash.Hex()))
}

// FilterInvalidDestinationCallers returns true if
// there is no dest caller, or if we are the dest caller for the specified domain
// and false otherwise, because relaying the message will fail
func (m MessageState) FilterInvalidDestinationCallers(cfg *config.Config) bool {
	zeroByteArr := make([]byte, 32)
	bech32DestinationCaller, err := DecodeDestinationCaller(m.DestinationCaller)
	if err != nil {
		return false
	}
	if !bytes.Equal(m.DestinationCaller, zeroByteArr) &&
		bech32DestinationCaller != cfg.Networks.Destination.Noble.MinterAddress {
		return false
	}
	return true
}

// FilterDisabledCCTPRoutes returns true if we have enabled relaying
// from a source domain to a destination domain, and false otherwise
func (m MessageState) FilterDisabledCCTPRoutes(cfg *config.Config) bool {
	val, ok := cfg.EnabledRoutes[m.DestDomain]
	return ok && val == m.DestDomain
}

// left padded input -> bech32 output
func DecodeDestinationCaller(input []byte) (string, error) {
	output, err := bech32.ConvertAndEncode("noble", input[12:])
	if err != nil {
		return "", errors.New("unable to encode destination caller")
	}
	return output, nil
}

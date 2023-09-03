package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
)

// Message defines ...
// https://github.com/circlefin/evm-cctp-contracts/blob/d53f0e1937a0a5c5158d356b6767b77dc32dcc90/src/messages/Message.sol#L29-L37
type Message struct {
	Version           uint32
	SourceDomain      uint32
	DestinationDomain uint32
	Nonce             uint64
	Sender            []byte
	Recipient         []byte
	DestinationCaller []byte
	MessageBody       []byte
}

// BurnMessage defines ...
// https://github.com/circlefin/evm-cctp-contracts/blob/d53f0e1937a0a5c5158d356b6767b77dc32dcc90/src/messages/BurnMessage.sol#L24-L29
type BurnMessage struct {
	Version       uint32
	BurnToken     []byte
	MintRecipient []byte
	Amount        *big.Int
	MessageSender []byte
}

// MetadataMessage defines ...
type MetadataMessage struct {
	Nonce     uint64
	Sender    []byte
	Channel   uint64
	Prefix    string
	Recipient []byte
	Memo      string
}

//

func (msg *Message) Parse(bz []byte) (*Message, error) {
	const (
		VersionIndex           = 0
		SourceDomainIndex      = 4
		DestinationDomainIndex = 8
		NonceIndex             = 12
		SenderIndex            = 20
		RecipientIndex         = 52
		DestinationCallerIndex = 84
		MessageBodyIndex       = 116
	)

	if len(bz) < MessageBodyIndex {
		return nil, errors.New("")
	}

	msg.Version = binary.BigEndian.Uint32(bz[VersionIndex:SourceDomainIndex])
	msg.SourceDomain = binary.BigEndian.Uint32(bz[SourceDomainIndex:DestinationDomainIndex])
	msg.DestinationDomain = binary.BigEndian.Uint32(bz[DestinationDomainIndex:NonceIndex])
	msg.Nonce = binary.BigEndian.Uint64(bz[NonceIndex:SenderIndex])
	msg.Sender = bz[SenderIndex:RecipientIndex]
	msg.Recipient = bz[RecipientIndex:DestinationCallerIndex]
	msg.DestinationCaller = bz[DestinationCallerIndex:MessageBodyIndex]
	msg.MessageBody = bz[MessageBodyIndex:]

	return msg, nil
}

func (c *BurnMessage) Parse(bz []byte) (*BurnMessage, error) {
	const (
		VersionIndex       = 0
		BurnTokenIndex     = 4
		MintRecipientIndex = 36
		AmountIndex        = 68
		MsgSenderIndex     = 100
		BurnContentLength  = 132
	)

	if len(bz) != BurnContentLength {
		return nil, errors.New("")
	}

	c.Version = binary.BigEndian.Uint32(bz[VersionIndex:BurnTokenIndex])
	c.BurnToken = bz[BurnTokenIndex:MintRecipientIndex]
	c.MintRecipient = bz[MintRecipientIndex:AmountIndex]
	c.Amount = new(big.Int).SetBytes(bz[AmountIndex:MsgSenderIndex])
	c.MessageSender = bz[MsgSenderIndex:]

	return c, nil
}

func (c *MetadataMessage) Parse(bz []byte) (*MetadataMessage, error) {
	const (
		NonceIndex     = 0
		SenderIndex    = 8
		ChannelIndex   = 40
		PrefixIndex    = 48
		RecipientIndex = 80
		MemoIndex      = 112
	)

	if len(bz) < MemoIndex {
		return nil, errors.New("")
	}

	c.Nonce = binary.BigEndian.Uint64(bz[NonceIndex:SenderIndex])
	c.Sender = bz[SenderIndex:ChannelIndex]
	c.Channel = binary.BigEndian.Uint64(bz[ChannelIndex:PrefixIndex])
	c.Prefix = string(bytes.TrimLeft(bz[PrefixIndex:RecipientIndex], string(byte(0))))
	c.Recipient = bz[RecipientIndex:MemoIndex]
	c.Memo = string(bz[MemoIndex:])

	return c, nil
}

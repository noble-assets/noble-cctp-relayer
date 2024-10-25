package solana

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"

	"github.com/cosmos/btcutil/base58"

	"github.com/strangelove-ventures/noble-cctp-relayer/solana/generated/message_transmitter"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// Instruction is a utility type to store accounts and data for a specific instruction.
type Instruction struct {
	Accounts []*solana.AccountMeta
	Data     []byte
}

// ParseTransaction is a utility that fetches a transaction from the Solana RPC
// and returns all messages sent via CCTP. It does not apply any filtering.
// TODO: Handle parsing of more than one message!
func (s *Solana) ParseTransaction(ctx context.Context, hash string) (*types.MessageState, error) {
	tx, err := s.GetTransaction(ctx, s.endpoints.RPC, hash)
	if err != nil {
		return nil, err
	}

	allAccounts := make(map[string]*solana.AccountMeta)
	for _, account := range tx.Result.Transaction.Message.AccountKeys {
		allAccounts[account.PubKey] = &solana.AccountMeta{
			PublicKey:  solana.MustPublicKeyFromBase58(account.PubKey),
			IsWritable: account.Writable,
			IsSigner:   account.Signer,
		}
	}

	var instructions []Instruction
	for _, tmp := range tx.Result.Meta.InnerInstructions {
		for _, instruction := range tmp.Instructions {
			if instruction.Accounts != nil && instruction.Data != "" {
				var accounts []*solana.AccountMeta
				for _, account := range instruction.Accounts {
					accounts = append(accounts, allAccounts[account])
				}

				instructions = append(instructions, Instruction{
					Accounts: accounts,
					Data:     base58.Decode(instruction.Data),
				})
			}
		}
	}

	prefix := fmt.Sprintf("Program return: %s ", s.messageTransmitter.String())
	for _, rawInstruction := range instructions {
		decodedInstruction, err := message_transmitter.DecodeInstruction(rawInstruction.Accounts, rawInstruction.Data)
		if err == nil {
			switch instruction := decodedInstruction.Impl.(type) {
			case *message_transmitter.SendMessage:
				for _, tmp := range tx.Result.Meta.LogMessages {
					if strings.HasPrefix(tmp, prefix) {
						bz, err := base64.StdEncoding.DecodeString(strings.Split(tmp, prefix)[1])
						if err != nil {
							return nil, err
						}
						nonce := binary.LittleEndian.Uint64(bz)

						message := cctptypes.Message{
							SourceDomain:      uint32(s.Domain()),
							DestinationDomain: instruction.Params.DestinationDomain,
							Nonce:             nonce,
							Sender:            instruction.GetSenderProgramAccount().PublicKey.Bytes(),
							Recipient:         instruction.Params.Recipient.Bytes(),
							DestinationCaller: make([]byte, 32),
							MessageBody:       instruction.Params.MessageBody,
						}

						bz, err = message.Bytes()
						if err != nil {
							return nil, err
						}

						return &types.MessageState{
							IrisLookupID: hex.EncodeToString(crypto.Keccak256(bz)),
							Status:       types.Created,
							SourceDomain: s.Domain(),
							DestDomain:   types.Domain(instruction.Params.DestinationDomain),
							SourceTxHash: hash,
							MsgSentBytes: bz,
							MsgBody:      message.MessageBody,
							Created:      time.Now(),
							Updated:      time.Now(),
							Nonce:        nonce,
						}, nil
					}
				}
			case *message_transmitter.SendMessageWithCaller:
				for _, tmp := range tx.Result.Meta.LogMessages {
					if strings.HasPrefix(tmp, prefix) {
						bz, err := base64.StdEncoding.DecodeString(strings.Split(tmp, prefix)[1])
						if err != nil {
							return nil, err
						}
						nonce := binary.LittleEndian.Uint64(bz)

						message := cctptypes.Message{
							SourceDomain:      uint32(s.Domain()),
							DestinationDomain: instruction.Params.DestinationDomain,
							Nonce:             nonce,
							Sender:            instruction.GetSenderProgramAccount().PublicKey.Bytes(),
							Recipient:         instruction.Params.Recipient.Bytes(),
							DestinationCaller: instruction.Params.DestinationCaller.Bytes(),
							MessageBody:       instruction.Params.MessageBody,
						}

						bz, err = message.Bytes()
						if err != nil {
							return nil, err
						}

						return &types.MessageState{
							IrisLookupID: hex.EncodeToString(crypto.Keccak256(bz)),
							Status:       types.Created,
							SourceDomain: s.Domain(),
							DestDomain:   types.Domain(instruction.Params.DestinationDomain),
							SourceTxHash: hash,
							MsgSentBytes: bz,
							MsgBody:      message.MessageBody,
							Created:      time.Now(),
							Updated:      time.Now(),
							Nonce:        nonce,
						}, nil
					}
				}
			}
		}
	}

	return nil, nil //nolint:nilnil
}

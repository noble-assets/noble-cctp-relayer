package solana

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"time"

	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/binary"
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
func (s *Solana) ParseTransaction(ctx context.Context, endpoint string, hash string) (events []types.MessageState, err error) {
	tx, err := s.GetTransaction(ctx, endpoint, hash)
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

	for _, rawInstruction := range instructions {
		decodedInstruction, err := message_transmitter.DecodeInstruction(rawInstruction.Accounts, rawInstruction.Data)
		if err == nil {
			switch instruction := decodedInstruction.Impl.(type) {
			case *message_transmitter.SendMessage:
			case *message_transmitter.SendMessageWithCaller:
				account := instruction.GetMessageSentEventDataAccount().PublicKey.String()
				info, err := s.GetAccountInfo(ctx, endpoint, account)
				if err != nil {
					return nil, err
				}

				bz, err := base64.StdEncoding.DecodeString(info.Result.Value.Data[0])
				if err != nil {
					return nil, err
				}

				var event message_transmitter.MessageSent
				err = event.UnmarshalWithDecoder(bin.NewBorshDecoder(bz))
				if err != nil {
					return nil, err
				}

				msg, err := new(cctptypes.Message).Parse(event.Message)
				if err != nil {
					return nil, err
				}

				events = append(events, types.MessageState{
					IrisLookupID:      hex.EncodeToString(crypto.Keccak256(event.Message)),
					Status:            types.Created,
					SourceDomain:      types.Domain(msg.SourceDomain),
					DestDomain:        types.Domain(msg.DestinationDomain),
					SourceTxHash:      hash,
					MsgSentBytes:      event.Message,
					MsgBody:           msg.MessageBody,
					DestinationCaller: msg.DestinationCaller,
					Created:           time.Now(),
					Updated:           time.Now(),
					Nonce:             msg.Nonce,
				})
			}
		}
	}

	return
}

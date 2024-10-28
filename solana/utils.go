package solana

import (
	"strconv"

	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
	"github.com/gagliardetto/solana-go"

	"cosmossdk.io/math"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// GetReceiveMessageAccounts returns all accounts to be included in a Receive
// Message instruction on the CCTP Message Transmitter program.
func (s *Solana) GetReceiveMessageAccounts(input *types.MessageState) []*solana.AccountMeta {
	accounts := make([]*solana.AccountMeta, 19)

	// 1 - Payer
	accounts[0] = &solana.AccountMeta{
		PublicKey:  s.wallet.PublicKey(),
		IsWritable: true,
		IsSigner:   true,
	}

	// 2 - Caller
	accounts[1] = &solana.AccountMeta{
		PublicKey:  s.wallet.PublicKey(),
		IsWritable: true,
		IsSigner:   true,
	}

	// 3 - Authority Pda
	authorityPda, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("message_transmitter_authority"),
		s.tokenMessengerMinter.Bytes(),
	}, s.messageTransmitter)

	accounts[2] = &solana.AccountMeta{
		PublicKey: authorityPda,
	}

	// 4 - Message Transmitter
	messageTransmitter, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("message_transmitter"),
	}, s.messageTransmitter)

	accounts[3] = &solana.AccountMeta{
		PublicKey: messageTransmitter,
	}

	// 5 - Used Nonces
	firstNonce := math.NewUint(input.Nonce).SubUint64(1).QuoUint64(6400).MulUint64(6400).AddUint64(1)
	usedNonces, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("used_nonces"),
		[]byte(strconv.Itoa(int(input.SourceDomain))),
		[]byte(firstNonce.String()),
	}, s.messageTransmitter)

	accounts[4] = &solana.AccountMeta{
		PublicKey:  usedNonces,
		IsWritable: true,
	}

	// 6 - Receiver
	accounts[5] = &solana.AccountMeta{
		PublicKey: s.tokenMessengerMinter,
	}

	// 7 - System Program
	accounts[6] = &solana.AccountMeta{
		PublicKey: solana.SystemProgramID,
	}

	// 8 - Event Authority
	eventAuthority, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("__event_authority"),
	}, s.messageTransmitter)

	accounts[7] = &solana.AccountMeta{
		PublicKey: eventAuthority,
	}

	// 9 - Program
	accounts[8] = &solana.AccountMeta{
		PublicKey: s.messageTransmitter,
	}

	// 10 - Token Messenger
	tokenMessenger, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("token_messenger"),
	}, s.tokenMessengerMinter)

	accounts[9] = &solana.AccountMeta{
		PublicKey: tokenMessenger,
	}

	// 11 - Remote Token Messenger
	remoteTokenMessenger, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("remote_token_messenger"),
		[]byte(strconv.Itoa(int(input.SourceDomain))),
	}, s.tokenMessengerMinter)

	accounts[10] = &solana.AccountMeta{
		PublicKey: remoteTokenMessenger,
	}

	// 12 - Token Minter
	tokenMinter, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("token_minter"),
	}, s.tokenMessengerMinter)

	accounts[11] = &solana.AccountMeta{
		PublicKey:  tokenMinter,
		IsWritable: true,
	}

	// 13 - Local Token
	localToken, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("local_token"),
		s.fiatToken.Bytes(),
	}, s.tokenMessengerMinter)

	accounts[12] = &solana.AccountMeta{
		PublicKey:  localToken,
		IsWritable: true,
	}

	// 14 - Token Pair
	tokenPair, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("token_pair"),
		[]byte(strconv.Itoa(int(input.SourceDomain))),
		s.remoteTokens[input.SourceDomain].Bytes(),
	}, s.tokenMessengerMinter)

	accounts[13] = &solana.AccountMeta{
		PublicKey: tokenPair,
	}

	// 15 - Recipient Token Account
	body, _ := new(cctptypes.BurnMessage).Parse(input.MsgBody)
	recipientTokenAccount := solana.PublicKeyFromBytes(body.MintRecipient)

	accounts[14] = &solana.AccountMeta{
		PublicKey:  recipientTokenAccount,
		IsWritable: true,
	}

	// 16 - Custody Token Account
	custodyTokenAccount, _, _ := solana.FindProgramAddress([][]byte{
		[]byte("custody"),
		s.fiatToken.Bytes(),
	}, s.tokenMessengerMinter)

	accounts[15] = &solana.AccountMeta{
		PublicKey:  custodyTokenAccount,
		IsWritable: true,
	}

	// 17 - Token Program
	accounts[16] = &solana.AccountMeta{
		PublicKey: solana.TokenProgramID,
	}

	// 18 - Event Authority
	eventAuthority, _, _ = solana.FindProgramAddress([][]byte{
		[]byte("__event_authority"),
	}, s.tokenMessengerMinter)

	accounts[17] = &solana.AccountMeta{
		PublicKey: eventAuthority,
	}

	// 19 - Program
	accounts[18] = &solana.AccountMeta{
		PublicKey: s.tokenMessengerMinter,
	}

	return accounts
}

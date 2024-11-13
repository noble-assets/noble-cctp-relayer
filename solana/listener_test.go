package solana_test

import (
	"context"
	"testing"

	extsolana "github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/solana"
)

// TestParseTransaction tests the transaction parsing utility used by the relayer.
// It fetches the first mainnet transfer from Solana to Noble.
//
// https://solscan.io/tx/4XhuTtTHxNFDfGn6A7ngvqT2dNoxRQFK7kpZwNY25gxXU5SPFCAk6ihj9JJdq5g7UMb7MSwyTt5r3TJbM4RgMVyJ
// https://www.mintscan.io/noble/tx/98C4BB3B29FBA6EBA3B14C3DFA85925C8223A88E268CBA8FBEAD6E5F3F9333D9
func TestParseTransaction(t *testing.T) {
	// ARRANGE: Create a new instance of Solana.
	key, err := extsolana.NewRandomPrivateKey()
	require.NoError(t, err)

	chain := solana.NewSolana(solana.Config{
		RPC: "https://corie-nhz8jx-fast-mainnet.helius-rpc.com",
		WS:  "",

		MessageTransmitter:   "CCTPmbSD7gX1bxKPAmg77w8oFzNFpaQiQUWD43TKaecd",
		TokenMessengerMinter: "CCTPiPYPc6AsJuwueEnWgSgucamXDZwBd53dQ11YiKX3",
		FiatToken:            "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",

		PrivateKey: key.String(),
	})
	require.NoError(t, chain.InitializeClients(context.Background(), log.NewNopLogger()))

	// ACT: Attempt to fetch and parse a transaction.
	event, err := chain.ParseTransaction(
		context.Background(),
		"4XhuTtTHxNFDfGn6A7ngvqT2dNoxRQFK7kpZwNY25gxXU5SPFCAk6ihj9JJdq5g7UMb7MSwyTt5r3TJbM4RgMVyJ",
	)
	// ASSERT: The action should've succeeded, and returned on message.
	require.NoError(t, err)
	require.NotNil(t, event)
}

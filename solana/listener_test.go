package solana_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/noble-cctp-relayer/solana"
)

// TestParseTransaction tests the transaction parsing utility used by the relayer.
// It fetches the first mainnet transfer from Solana to Noble.
//
// https://solscan.io/tx/4XhuTtTHxNFDfGn6A7ngvqT2dNoxRQFK7kpZwNY25gxXU5SPFCAk6ihj9JJdq5g7UMb7MSwyTt5r3TJbM4RgMVyJ
// https://www.mintscan.io/noble/tx/98C4BB3B29FBA6EBA3B14C3DFA85925C8223A88E268CBA8FBEAD6E5F3F9333D9
func TestParseTransaction(t *testing.T) {
	// ACT: Attempt to fetch and parse a transaction.
	events, err := new(solana.Solana).ParseTransaction(
		"https://corie-nhz8jx-fast-mainnet.helius-rpc.com",
		"4XhuTtTHxNFDfGn6A7ngvqT2dNoxRQFK7kpZwNY25gxXU5SPFCAk6ihj9JJdq5g7UMb7MSwyTt5r3TJbM4RgMVyJ",
	)
	// ASSERT: The action should've succeeded, and returned on message.
	require.NoError(t, err)
	require.Len(t, events, 1)
}

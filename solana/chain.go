package solana

import "github.com/strangelove-ventures/noble-cctp-relayer/types"

// TODO: Uncomment this once the full interface is implemented!
// var _ types.Chain = (*Solana)(nil)

type Solana struct{}

// Name implements the types.Chain interface.
func (s *Solana) Name() string { return "Solana" }

// Domain returns the specific domain for Solana.
// https://developers.circle.com/stablecoins/supported-domains
func (s *Solana) Domain() types.Domain { return 5 }

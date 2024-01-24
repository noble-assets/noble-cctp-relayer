package ethereum

import (
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.ChainConfig = (*ChainConfig)(nil)

type ChainConfig struct {
	RPC                string `yaml:"rpc"`
	WS                 string `yaml:"ws"`
	domain             types.Domain
	ChainID            int64  `yaml:"chain-id"`
	MessageTransmitter string `yaml:"message-transmitter"`

	StartBlock     uint64 `yaml:"start-block"`
	LookbackPeriod uint64 `yaml:"lookback-period"`

	BroadcastRetries       int `yaml:"broadcast-retries"`
	BroadcastRetryInterval int `yaml:"broadcast-retry-interval"`

	// TODO move to keyring
	MinterPrivateKey string `yaml:"minter-private-key"`
}

func (c *ChainConfig) Chain(name string) (types.Chain, error) {
	return NewChain(
		name,
		c.domain,
		c.ChainID,
		c.RPC,
		c.WS,
		c.MessageTransmitter,
		c.StartBlock,
		c.LookbackPeriod,
		c.MinterPrivateKey,
		c.BroadcastRetries,
		c.BroadcastRetryInterval,
	)
}
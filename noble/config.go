package noble

import "github.com/strangelove-ventures/noble-cctp-relayer/types"

var _ types.ChainConfig = (*ChainConfig)(nil)

const defaultBlockQueueChannelSize = 1000000

type ChainConfig struct {
	RPC     string `yaml:"rpc"`
	ChainID string `yaml:"chain-id"`

	StartBlock     uint64 `yaml:"start-block"`
	LookbackPeriod uint64 `yaml:"lookback-period"`
	Workers        uint32 `yaml:"workers"`

	TxMemo                 string `yaml:"tx-memo"`
	GasLimit               uint64 `yaml:"gas-limit"`
	BroadcastRetries       int    `yaml:"broadcast-retries"`
	BroadcastRetryInterval int    `yaml:"broadcast-retry-interval"`

	BlockQueueChannelSize uint64 `yaml:"block-queue-channel-size"`

	MinAmount uint64 `yaml:"min-amount"`

	// TODO move to keyring
	MinterPrivateKey string `yaml:"minter-private-key"`
}

func (c *ChainConfig) Chain(name string) (types.Chain, error) {
	return NewChain(
		c.RPC,
		c.ChainID,
		c.MinterPrivateKey,
		c.StartBlock,
		c.LookbackPeriod,
		c.Workers,
		c.GasLimit,
		c.TxMemo,
		c.BroadcastRetries,
		c.BroadcastRetryInterval,
		c.BlockQueueChannelSize,
		c.MinAmount,
	)
}

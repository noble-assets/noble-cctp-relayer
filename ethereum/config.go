package ethereum

import (
	"fmt"
	"os"
	"strings"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.ChainConfig = (*ChainConfig)(nil)

type ChainConfig struct {
	RPC                string `yaml:"rpc"`
	WS                 string `yaml:"ws"`
	Domain             types.Domain
	ChainID            int64  `yaml:"chain-id"`
	MessageTransmitter string `yaml:"message-transmitter"`

	StartBlock     uint64 `yaml:"start-block"`
	LookbackPeriod uint64 `yaml:"lookback-period"`

	BroadcastRetries       int `yaml:"broadcast-retries"`
	BroadcastRetryInterval int `yaml:"broadcast-retry-interval"`

	MinMintAmount uint64 `yaml:"min-mint-amount"`

	MetricsDenom    string `yaml:"metrics-denom"`
	MetricsExponent int    `yaml:"metrics-exponent"`

	MinterPrivateKey string `yaml:"minter-private-key"`
}

func (c *ChainConfig) Chain(name string) (types.Chain, error) {
	if len(c.MinterPrivateKey) == 0 {
		envKey := strings.ToUpper(name) + "_PRIV_KEY"
		privKey := os.Getenv(envKey)
		if len(privKey) == 0 {
			return nil, fmt.Errorf("env variable %s is empty, priv key not found for chain %s", envKey, name)
		} else {
			c.MinterPrivateKey = privKey
		}
	}

	return NewChain(
		name,
		c.Domain,
		c.ChainID,
		c.RPC,
		c.WS,
		c.MessageTransmitter,
		c.StartBlock,
		c.LookbackPeriod,
		c.MinterPrivateKey,
		c.BroadcastRetries,
		c.BroadcastRetryInterval,
		c.MinMintAmount,
		c.MetricsDenom,
		c.MetricsExponent,
	)
}

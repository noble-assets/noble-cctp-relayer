package noble

import (
	"fmt"
	"os"
	"strings"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

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

	MinMintAmount uint64 `yaml:"min-mint-amount"`

	MinterPrivateKey string `yaml:"minter-private-key"`
}

func (c *ChainConfig) Chain(name string) (types.Chain, error) {
	envKey := strings.ToUpper(name) + "_PRIV_KEY"
	privKey := os.Getenv(envKey)

	if len(c.MinterPrivateKey) == 0 || len(privKey) != 0 {
		if len(privKey) == 0 {
			return nil, fmt.Errorf("env variable %s is empty, priv key not found for chain %s", envKey, name)
		} else {
			c.MinterPrivateKey = privKey
		}
	}

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
		c.MinMintAmount,
	)
}

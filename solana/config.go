package solana

import "github.com/strangelove-ventures/noble-cctp-relayer/types"

var _ types.ChainConfig = (*Config)(nil)

type Config struct {
	RPC string `yaml:"rpc"`
	WS  string `yaml:"ws"`

	MessageTransmitter string `yaml:"message-transmitter"`
}

func (cfg Config) Chain(_ string) (types.Chain, error) {
	return NewSolana(cfg.RPC, cfg.WS, cfg.MessageTransmitter), nil
}

package solana

import "github.com/strangelove-ventures/noble-cctp-relayer/types"

var _ types.ChainConfig = (*Config)(nil)

type Config struct {
	RPC string `yaml:"rpc"`
	WS  string `yaml:"ws"`

	MessageTransmitter   string                  `yaml:"message-transmitter"`
	TokenMessengerMinter string                  `yaml:"token-messenger-minter"`
	FiatToken            string                  `yaml:"fiat-token"`
	RemoteTokens         map[types.Domain]string `json:"remote-tokens"`

	PrivateKey string `yaml:"private-key"`
}

func (cfg Config) Chain(_ string) (types.Chain, error) {
	return NewSolana(cfg), nil
}

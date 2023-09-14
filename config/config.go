package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Networks struct {
		Source struct {
			Ethereum struct {
				DomainId            uint32   `yaml:"domain_id"`
				RPC                 string   `yaml:"rpc"`
				MessageTransmitter  string   `yaml:"message_transmitter"`
				ValidTokenAddresses []string `yaml:"valid_token_addresses"`
				RequestQueueSize    uint32   `yaml:"request_queue_size"`
				StartBlock          uint64   `yaml:"start_block"`
				LookbackPeriod      uint64   `yaml:"lookback_period"`
				Enabled             bool     `yaml:"enabled"`
			} `yaml:"ethereum"`
		} `yaml:"source"`
		Destination struct {
			Noble struct {
				DomainId                   uint32   `yaml:"domain_id"`
				API                        string   `yaml:"api"`
				RPC                        string   `yaml:"rpc"`
				ChainId                    string   `yaml:"chain_id"`
				GasLimit                   uint64   `yaml:"gas_limit"`
				BroadcastRetries           int      `yaml:"broadcast_retries"`
				BroadcastRetryInterval     int      `yaml:"broadcast_retry_interval"`
				FilterForwardsByIbcChannel bool     `yaml:"filter_forwards_by_ibc_channel"`
				ForwardingChannelWhitelist []string `yaml:"forwarding_channel_whitelist"`
			} `yaml:"noble"`
		} `yaml:"destination"`
	} `yaml:"networks"`
	EnabledRoutes map[uint32]uint32 `yaml:"enabled_routes"`
	Minters       map[uint32]struct {
		MinterAddress    string `yaml:"minter_address"`
		MinterPrivateKey string `yaml:"minter_private_key"`
	} `yaml:"enabled_routes"`
	AttestationBaseUrl string `yaml:"attestation_base_url"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)

	return
}

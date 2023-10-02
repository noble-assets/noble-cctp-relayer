package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Networks struct {
		Source struct {
			Ethereum struct {
				DomainId           uint32 `yaml:"domain-id"`
				RPC                string `yaml:"rpc"`
				MessageTransmitter string `yaml:"message-transmitter"`
				RequestQueueSize   uint32 `yaml:"request-queue-size"`
				StartBlock         uint64 `yaml:"start-block"`
				LookbackPeriod     uint64 `yaml:"lookback-period"`
				Enabled            bool   `yaml:"enabled"`
			} `yaml:"ethereum"`
		} `yaml:"source"`
		Destination struct {
			Noble struct {
				DomainId                   uint32   `yaml:"domain-id"`
				RPC                        string   `yaml:"rpc"`
				API                        string   `yaml:"api"`
				ChainId                    string   `yaml:"chain-id"`
				GasLimit                   uint64   `yaml:"gas-limit"`
				BroadcastRetries           int      `yaml:"broadcast-retries"`
				BroadcastRetryInterval     int      `yaml:"broadcast-retry-interval"`
				FilterForwardsByIbcChannel bool     `yaml:"filter-forwards-by-ibc-channel"`
				ForwardingChannelWhitelist []string `yaml:"forwarding-channel-whitelist"`
			} `yaml:"noble"`
		} `yaml:"destination"`
		EnabledRoutes map[uint32]uint32 `yaml:"enabled-routes"`
		Minters       map[uint32]struct {
			MinterAddress    string `yaml:"minter-address"`
			MinterPrivateKey string `yaml:"minter-private-key"`
		} `yaml:"minters"`
	} `yaml:"networks"`
	Circle struct {
		AttestationBaseUrl string `yaml:"attestation-base-url"`
		FetchRetries       int    `yaml:"fetch-retries"`
		FetchRetryInterval int    `yaml:"fetch-retry-interval"`
	} `yaml:"circle"`
	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
	Api                  struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)
	return
}

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Networks struct {
		Ethereum struct {
			DomainId            uint32   `yaml:"domain_id"`
			RPC                 string   `yaml:"rpc"`
			MessageTransmitter  string   `yaml:"message_transmitter"`
			ValidTokenAddresses []string `yaml:"valid_token_addresses"`
		} `yaml:"ethereum"`
		Noble struct {
			DomainId uint32 `yaml:"domain_id"`
			RPC      string `yaml:"rpc"`
			ChainId  string `yaml:"chain_id"`
			PrivKey  string `yaml:"priv_key"`
			GasLimit uint64 `yaml:"gas_limit"`
		} `yaml:"noble"`
	} `yaml:"networks"`

	AttestationBaseUrl string `yaml:"attestation_base_url"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)

	return
}

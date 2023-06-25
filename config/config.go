package config

import (
	"os"

	"cosmossdk.io/log"
	"gopkg.in/yaml.v3"
)

var logger = log.NewLogger(os.Stdout)

type Config struct {
	Networks struct {
		Ethereum struct {
			RPC                string `yaml:"rpc"`
			TokenMessenger     string `yaml:"token_messenger"`
			MessageTransmitter string `yaml:"message_transmitter"`
		} `yaml:"ethereum"`
		Noble struct {
			RPC           string `yaml:"rpc"`
			DestinationId uint32 `yaml:"destination_id"`
		} `yaml:"noble"`
	} `yaml:"networks"`
	Indexer struct {
		StartBlock          int64    `yaml:"start_block"`
		AttestationBaseUrl  string   `yaml:"attestation_base_url"`
		ValidTokenAddresses []string `yaml:"valid_token_addresses"`
	} `yaml:"indexer"`
}

func Parse(file string) (cfg Config) {
	if file == "" {
		logger.Error("no configuration provided")
		os.Exit(1)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		logger.Error("could not read configuration", "err", err)
		os.Exit(1)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error("could not parse configuration", "err", err)
		os.Exit(1)
	}

	logger.Info("successfully parsed config file")
	return
}

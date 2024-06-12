package integration_test

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Wallets used for integration testing
type IntegrationWallets struct {
	Networks struct {
		Ethereum struct {
			Address    string `yaml:"address"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"ethereum"`
		Noble struct {
			Address    string `yaml:"address"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"noble"`
	} `yaml:"networks"`
}

type IntegrationConfig struct {
	Testnet map[string]*IntegrationChain `yaml:"testnet"`
	Mainnet map[string]*IntegrationChain `yaml:"mainnet"`
}

type IntegrationChain struct {
	ChainID               string `yaml:"chain-id"`
	Domain                uint32 `yaml:"domain"`
	Address               string `yaml:"address"`
	PrivateKey            string `yaml:"private-key"`
	RPC                   string `yaml:"rpc"`
	UsdcTokenAddress      string `yaml:"usdc-token-address"`
	TokenMessengerAddress string `yaml:"token-messenger-address"`
	DestinationCaller     string `yaml:"destination-caller"`
}

func ParseIntegration(file string) (*IntegrationConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var integrationConfig *IntegrationConfig
	err = yaml.Unmarshal(data, &integrationConfig)
	if err != nil {
		return nil, err
	}

	return integrationConfig, nil
}

package integration_testing

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Networks struct {
		Ethereum struct {
			RPC        string `yaml:"rpc"`
			PrivateKey string `yaml:"private_key"`
			FaucetUrl  string `yaml:"faucet_url"`
		} `yaml:"ethereum"`
		Noble struct {
			RPC       string `yaml:"rpc"`
			FaucetUrl string `yaml:"faucet_url"`
		} `yaml:"noble"`
	} `yaml:"networks"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)

	return
}

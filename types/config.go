package types

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Chains        map[string]ChainConfig `yaml:"chains"`
	EnabledRoutes map[Domain]Domain      `yaml:"enabled-routes"`
	Circle        struct {
		AttestationBaseUrl string `yaml:"attestation-base-url"`
		FetchRetries       int    `yaml:"fetch-retries"`
		FetchRetryInterval int    `yaml:"fetch-retry-interval"`
	} `yaml:"circle"`
	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
	Api                  struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
}

func Parse(file string) (cfg Config, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}

type ChainConfig interface {
	Chain(name string) (Chain, error)
}

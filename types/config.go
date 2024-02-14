package types

type Config struct {
	Chains        map[string]ChainConfig `yaml:"chains"`
	EnabledRoutes map[Domain][]Domain    `yaml:"enabled-routes"`
	Circle        struct {
		AttestationBaseUrl string `yaml:"attestation-base-url"`
		FetchRetries       int    `yaml:"fetch-retries"`
		FetchRetryInterval int    `yaml:"fetch-retry-interval"`
	} `yaml:"circle"`
	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
	Api                  struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
	MinAmount uint64 `yaml:"min-amount"`
}

type ConfigWrapper struct {
	Chains        map[string]map[string]any `yaml:"chains"`
	EnabledRoutes map[Domain][]Domain       `yaml:"enabled-routes"`
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

type ChainConfig interface {
	Chain(name string) (Chain, error)
}

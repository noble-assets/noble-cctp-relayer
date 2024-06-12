package types

type Config struct {
	Chains        map[string]ChainConfig `yaml:"chains"`
	EnabledRoutes map[Domain][]Domain    `yaml:"enabled-routes"`
	Circle        CircleSettings         `yaml:"circle"`

	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
	API                  struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
}

type ConfigWrapper struct {
	Chains        map[string]map[string]any `yaml:"chains"`
	EnabledRoutes map[Domain][]Domain       `yaml:"enabled-routes"`
	Circle        CircleSettings            `yaml:"circle"`

	ProcessorWorkerCount uint32 `yaml:"processor-worker-count"`
	API                  struct {
		TrustedProxies []string `yaml:"trusted-proxies"`
	} `yaml:"api"`
}

type CircleSettings struct {
	AttestationBaseURL string `yaml:"attestation-base-url"`
	FetchRetries       int    `yaml:"fetch-retries"`
	FetchRetryInterval int    `yaml:"fetch-retry-interval"`
}

type ChainConfig interface {
	Chain(name string) (Chain, error)
}

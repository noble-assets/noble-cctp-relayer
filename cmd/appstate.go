package cmd

import (
	"fmt"
	"os"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"gopkg.in/yaml.v2"
)

// appState is the modifiable state of the application.
type AppState struct {
	Config *types.Config

	ConfigPath string

	Debug bool

	Logger log.Logger
}

func NewappState() *AppState {
	return &AppState{}
}

func (a *AppState) InitLogger() {
	if a.Debug {
		a.Logger = log.NewLogger(os.Stdout)
	} else {
		a.Logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.InfoLevel))
	}
}

// loadConfigFile reads config file into a.Config if file is present.
func (a *AppState) loadConfigFile() error {
	config, err := ParseConfig(a.ConfigPath)
	if err != nil {
		a.Logger.Error("unable to parse config file", "location", a.ConfigPath, "err", err)
		os.Exit(1)
	}
	a.Logger.Info("successfully parsed config file", "location", a.ConfigPath)
	a.Config = config

	return nil
}

// ParseConfig parses the app config file
func ParseConfig(file string) (*types.Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %w", err)
	}

	var cfg types.ConfigWrapper
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	c := types.Config{
		EnabledRoutes:        cfg.EnabledRoutes,
		Circle:               cfg.Circle,
		ProcessorWorkerCount: cfg.ProcessorWorkerCount,
		Api:                  cfg.Api,
		Chains:               make(map[string]types.ChainConfig),
	}

	for name, chain := range cfg.Chains {
		yamlbz, err := yaml.Marshal(chain)
		if err != nil {
			return nil, err
		}

		switch name {
		case "noble":
			var cc noble.ChainConfig
			if err := yaml.Unmarshal(yamlbz, &cc); err != nil {
				return nil, err
			}
			c.Chains[name] = &cc
		default:
			var cc ethereum.ChainConfig
			if err := yaml.Unmarshal(yamlbz, &cc); err != nil {
				return nil, err
			}
			c.Chains[name] = &cc
		}
	}
	return &c, err
}

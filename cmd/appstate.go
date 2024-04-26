package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

const nobleChainName = "noble"

// appState is the modifiable state of the application.
type AppState struct {
	Config *types.Config

	ConfigPath string

	Debug bool

	LogLevel string

	Logger log.Logger
}

func NewAppState() *AppState {
	return &AppState{}
}

// InitAppState checks if a logger and config are present. If not, it adds them to the AppState
func (a *AppState) InitAppState() {
	if a.Logger == nil {
		a.InitLogger()
	}
	if a.Config == nil {
		a.loadConfigFile()
	}
}

func (a *AppState) InitLogger() {
	// info level is default
	level := zerolog.InfoLevel
	switch a.LogLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}

	// a.Debug overrides a.loglevel
	if a.Debug {
		a.Logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))
	} else {
		a.Logger = log.NewLogger(os.Stdout, log.LevelOption(level))
	}
}

// loadConfigFile loads a configuration into the AppState. It uses the AppState ConfigPath
// to determine file path to config.
func (a *AppState) loadConfigFile() {
	if a.Logger == nil {
		a.InitLogger()
	}
	config, err := ParseConfig(a.ConfigPath)
	if err != nil {
		a.Logger.Error("Unable to parse config file", "location", a.ConfigPath, "err", err)
		os.Exit(1)
	}
	a.Logger.Info("Successfully parsed config file", "location", a.ConfigPath)
	a.Config = config

	err = a.validateConfig()
	if err != nil {
		a.Logger.Error("Invalid config", "err", err)
		os.Exit(1)
	}
}

// validateConfig checks the AppState Config for any invalid settings.
func (a *AppState) validateConfig() error {
	// validate chains
	for name, cfg := range a.Config.Chains {
		// check if chain is noble
		if name == nobleChainName {
			// validate noble chain
			cc := cfg.(*noble.ChainConfig)
			err := a.validateChain(
				name,
				cc.ChainID,
				"",
				cc.RPC,
				"",
				cc.BroadcastRetries,
				cc.BroadcastRetryInterval,
				cc.MinMintAmount,
			)
			if err != nil {
				return err
			}
		} else {
			// validate eth based chains
			cc := cfg.(*ethereum.ChainConfig)
			err := a.validateChain(
				name,
				fmt.Sprintf("%d", cc.ChainID),
				fmt.Sprintf("%d", cc.Domain),
				cc.RPC,
				cc.WS,
				cc.BroadcastRetries,
				cc.BroadcastRetryInterval,
				cc.MinMintAmount,
			)
			if err != nil {
				return err
			}
		}
	}

	// ensure at least 1 enabled route
	if len(a.Config.EnabledRoutes) == 0 {
		return fmt.Errorf("at least one route must be enabled in the config")
	}

	// validate circle api config
	err := a.validateCircleConfig()
	if err != nil {
		return err
	}

	// validate processor worker count
	if a.Config.ProcessorWorkerCount == 0 {
		return fmt.Errorf("ProcessorWorkerCount must be greater than zero in the config")
	}

	return nil
}

// validateChain ensures the chain is configured correctly
func (a *AppState) validateChain(
	name string,
	chainID string,
	domain string,
	rpcURL string,
	wsURL string,
	broadcastRetries int,
	broadcastRetryInterval int,
	minMintAmount uint64,
) error {
	if name == "" {
		return fmt.Errorf("chain name must be set in the config")
	}

	if chainID == "" {
		return fmt.Errorf("chainID must be set in the config (chain: %s) (chainID: %s)", name, chainID)
	}

	// domain is hardcoded to 4 for noble chain
	if domain == "" && name != nobleChainName {
		return fmt.Errorf("domain must be set in the config (chain: %s) (domain: %s)", name, domain)
	}

	if rpcURL == "" {
		return fmt.Errorf("rpcURL must be set in the config (chain: %s) (rpcURL: %s)", name, rpcURL)
	}

	// we do not use a websocket for noble
	if wsURL == "" && name != nobleChainName {
		return fmt.Errorf("wsURL must be set in the config (chain: %s) (wsURL: %s)", name, wsURL)
	}

	if broadcastRetries <= 0 {
		return fmt.Errorf("broadcastRetries must be greater than zero in the config (chain: %s) (broadcastRetries: %d)", name, broadcastRetries)
	}

	if broadcastRetryInterval <= 0 {
		return fmt.Errorf("broadcastRetryInterval must be greater than zero in the config (chain: %s) (broadcastRetryInterval: %d)", name, broadcastRetryInterval)
	}

	// noble has free minting
	if minMintAmount == 0 && name != nobleChainName {
		return fmt.Errorf("ETH-based chains must have a minMintAmount greater than zero in the config (chain: %s) (minMintAmount: %d)", name, minMintAmount)
	}

	return nil
}

// validateCircleConfig ensures the circle api is configured correctly
func (a *AppState) validateCircleConfig() error {
	if a.Config.Circle.AttestationBaseURL == "" {
		return fmt.Errorf("AttestationBaseUrl is required in the config")
	}

	if a.Config.Circle.FetchRetryInterval == 0 {
		return fmt.Errorf("FetchRetryInterval must be greater than zero in the config")
	}

	return nil
}

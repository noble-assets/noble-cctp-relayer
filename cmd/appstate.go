package cmd

import (
	"os"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

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

}

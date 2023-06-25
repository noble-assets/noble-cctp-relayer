package cmd

import (
	"os"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
)

var logger log.Logger

var (
	cfg config.Config

	MessageTransmitter common.Address
	TokenMessenger     common.Address
	ValidTokens        = make(map[common.Address]bool)

	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "noble-cctp-relayer",
	Short: "A CLI tool for relaying CCTP messages to Noble",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	cobra.OnInitialize(func() {
		cfg = config.Parse(cfgFile)

		MessageTransmitter = common.HexToAddress(cfg.Networks.Ethereum.MessageTransmitter)
		TokenMessenger = common.HexToAddress(cfg.Networks.Ethereum.TokenMessenger)
		for _, token := range cfg.Indexer.ValidTokenAddresses {
			ValidTokens[common.HexToAddress(token)] = true
		}

		if verbose {
			logger = log.NewLogger(os.Stdout)
		} else {
			logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.InfoLevel))
		}
	})
}

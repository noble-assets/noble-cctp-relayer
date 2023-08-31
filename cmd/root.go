package cmd

import (
	"os"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
)

var logger log.Logger

var (
	cfg     config.Config
	cfgFile string
	verbose bool

	MessageTransmitter    common.Address
	MessageTransmitterABI abi.ABI
	MessageSent           abi.Event
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	rootCmd.AddCommand(startCmd)

	cobra.OnInitialize(func() {
		if verbose {
			logger = log.NewLogger(os.Stdout)
		} else {
			logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.InfoLevel))
		}

		cfg = config.Parse(cfgFile)
		logger.Info("successfully parsed config file", "location", cfgFile)

		MessageTransmitter = common.HexToAddress(cfg.Networks.Ethereum.MessageTransmitter)
	})
}

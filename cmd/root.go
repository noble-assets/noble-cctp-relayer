package cmd

import (
	"context"
	"os"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"

	"cosmossdk.io/log"
	"github.com/spf13/cobra"
)

var (
	Cfg     *types.Config
	cfgFile string
	verbose bool

	Logger log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "noble-cctp-relayer",
	Short: "A CLI tool for relaying CCTP messages",
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		Logger.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		Start(),
	)
}

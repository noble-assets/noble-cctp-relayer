package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

const appName = "noble-cctp-relayer"

func NewRootCmd() *cobra.Command {
	// Use a local app state instance scoped to the new root command,
	// so that tests don't concurrently access the state.
	a := &AppState{}

	var rootCmd = &cobra.Command{
		Use:   appName,
		Short: "A CLI tool for relaying CCTP messages",
	}

	// Inside persistent pre-run because this takes effect after flags are parsed.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {

		a.InitLogger()

		if err := a.loadConfigFile(); err != nil {
			return err
		}

		return nil
	}

	// flags
	rootCmd.PersistentFlags().StringVar(&a.ConfigPath, "config", "config.yaml", "file path of config file")
	rootCmd.PersistentFlags().BoolVarP(&a.Debug, "verbose", "v", false, "use this flag to set log level to `debug`")

	// Add commands
	rootCmd.AddCommand(
		Start(a),
	)

	return rootCmd
}

func Execute(ctx context.Context) {
	rootCmd := NewRootCmd()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

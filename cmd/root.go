package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

const appName = "noble-cctp-relayer"

var defaultConfigPath = "./config"

func NewRootCmd() *cobra.Command {
	// Use a local app state instance scoped to the new root command,
	// so that tests don't concurrently access the state.
	a := NewAppState()

	var rootCmd = &cobra.Command{
		Use:   appName,
		Short: "A CLI tool for relaying CCTP messages",
	}

	// Add commands
	rootCmd.AddCommand(
		Start(a),
		getVersionCmd(),
		configShowCmd(a),
	)

	addAppPersistantFlags(rootCmd, a)
	return rootCmd
}

func Execute(ctx context.Context) {
	rootCmd := NewRootCmd()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

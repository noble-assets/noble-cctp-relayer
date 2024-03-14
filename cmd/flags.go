package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	flagConfigPath  = "config"
	flagVerbose     = "verbose"
	flagLogLevel    = "log-level"
	flagJSON        = "json"
	flagMetricsPort = "metrics-port"
)

func addAppPersistantFlags(cmd *cobra.Command, a *AppState) *cobra.Command {
	cmd.PersistentFlags().StringVar(&a.ConfigPath, flagConfigPath, defaultConfigPath, "file path of config file")
	cmd.PersistentFlags().BoolVarP(&a.Debug, flagVerbose, "v", false, fmt.Sprintf("use this flag to set log level to `debug` (overrides %s flag)", flagLogLevel))
	cmd.PersistentFlags().StringVar(&a.LogLevel, flagLogLevel, "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().Int16P(flagMetricsPort, "p", 2112, "customize Prometheus metrics port")
	return cmd

}

func addJsonFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(flagJSON, false, "return in json format")
	return cmd
}

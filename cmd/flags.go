package cmd

import (
	"github.com/spf13/cobra"
)

const (
	flagConfigPath = "config"
	// depreciated
	flagVerbose  = "verbose"
	flagLogLevel = "log-level"
	flagJSON     = "json"
)

func addAppPersistantFlags(cmd *cobra.Command, a *AppState) *cobra.Command {
	cmd.PersistentFlags().StringVar(&a.ConfigPath, flagConfigPath, defaultConfigPath, "file path of config file")
	cmd.PersistentFlags().BoolVarP(&a.Debug, flagVerbose, "v", false, "use this flag to set log level to `debug`")
	cmd.PersistentFlags().MarkDeprecated(flagVerbose, "depericated")
	cmd.PersistentFlags().StringVar(&a.LogLevel, flagLogLevel, "info", "log level (debug, info, warn, error)")
	return cmd

}

func addJsonFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(flagJSON, false, "return in json format")
	return cmd
}

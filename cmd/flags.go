package cmd

import (
	"github.com/spf13/cobra"
)

const (
	flagConfigPath = "config"
	flagVerbose    = "verbose"
	flagJSON       = "json"
)

func addAppPersistantFlags(cmd *cobra.Command, a *AppState) *cobra.Command {
	cmd.PersistentFlags().StringVar(&a.ConfigPath, flagConfigPath, defaultConfigPath, "file path of config file")
	cmd.PersistentFlags().BoolVarP(&a.Debug, flagVerbose, "v", false, "use this flag to set log level to `debug`")
	return cmd

}

func addJsonFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(flagJSON, false, "return in json format")
	return cmd
}

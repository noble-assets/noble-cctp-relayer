package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	flagConfigPath     = "config"
	flagVerbose        = "verbose"
	flagLogLevel       = "log-level"
	flagJSON           = "json"
	flagMetricsAddress = "metrics-address"
	flagMetricsPort    = "metrics-port"
	flagFlushInterval  = "flush-interval"
	flagFlushOnlyMode  = "flush-only-mode"
)

func addAppPersistantFlags(cmd *cobra.Command, a *AppState) *cobra.Command {
	cmd.PersistentFlags().StringVar(&a.ConfigPath, flagConfigPath, defaultConfigPath, "file path of config file")
	cmd.PersistentFlags().BoolVarP(&a.Debug, flagVerbose, "v", false, fmt.Sprintf("use this flag to set log level to `debug` (overrides %s flag)", flagLogLevel))
	cmd.PersistentFlags().StringVar(&a.LogLevel, flagLogLevel, "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().String(flagMetricsAddress, "localhost", "customize Prometheus metrics host address, this can be used in conjunction with `metrics-port` to adjust entire endpoint")
	cmd.PersistentFlags().Int16P(flagMetricsPort, "p", 2112, "customize Prometheus metrics port")
	cmd.PersistentFlags().DurationP(flagFlushInterval, "i", 0, "how frequently should a flush routine be run")
	cmd.PersistentFlags().BoolP(flagFlushOnlyMode, "f", false, "only run the background flush routine (acts as a redundant relayer)")
	return cmd
}

func addJSONFlag(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(flagJSON, false, "return in json format")
	return cmd
}

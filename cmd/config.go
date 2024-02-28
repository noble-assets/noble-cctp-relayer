package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"gopkg.in/yaml.v2"
)

// Command for printing current configuration
func configShowCmd(a *AppState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "showConfig",
		Aliases: []string{"sc"},
		Short:   "Prints current configuration. By default it prints in yaml",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			a.InitAppState()
			return nil
		},
		Example: strings.TrimSpace(fmt.Sprintf(`
$ %s showConfig --config %s
$ %s sc`, appName, defaultConfigPath, appName)),
		RunE: func(cmd *cobra.Command, args []string) error {

			jsn, err := cmd.Flags().GetBool(flagJSON)
			if err != nil {
				return err
			}

			switch {
			case jsn:
				out, err := json.Marshal(a.Config)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			default:
				out, err := yaml.Marshal(a.Config)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}
		},
	}
	addJsonFlag(cmd)
	return cmd
}

// ParseConfig parses the app config file
func ParseConfig(file string) (*types.Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %w", err)
	}

	var cfg types.ConfigWrapper
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	c := types.Config{
		EnabledRoutes:        cfg.EnabledRoutes,
		Circle:               cfg.Circle,
		ProcessorWorkerCount: cfg.ProcessorWorkerCount,
		Api:                  cfg.Api,
		Chains:               make(map[string]types.ChainConfig),
	}

	for name, chain := range cfg.Chains {
		yamlbz, err := yaml.Marshal(chain)
		if err != nil {
			return nil, err
		}

		switch name {
		case "noble":
			var cc noble.ChainConfig
			if err := yaml.Unmarshal(yamlbz, &cc); err != nil {
				return nil, err
			}
			c.Chains[name] = &cc
		default:
			var cc ethereum.ChainConfig
			if err := yaml.Unmarshal(yamlbz, &cc); err != nil {
				return nil, err
			}
			c.Chains[name] = &cc
		}
	}
	return &c, err
}

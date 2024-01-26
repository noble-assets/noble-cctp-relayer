package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"gopkg.in/yaml.v2"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	rootCmd.AddCommand(startCmd)

	cobra.OnInitialize(func() {
		if verbose {
			Logger = log.NewLogger(os.Stdout)
		} else {
			Logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.InfoLevel))
		}

		var err error
		Cfg, err = Parse(cfgFile)
		if err != nil {
			Logger.Error("unable to parse config file", "location", cfgFile, "err", err)
			os.Exit(1)
		}
		Logger.Info("successfully parsed config file", "location", cfgFile)

		// start api server
		go startApi()
	})
}

func startApi() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	err := router.SetTrustedProxies(Cfg.Api.TrustedProxies) // vpn.primary.strange.love
	if err != nil {
		Logger.Error("unable to set trusted proxies on API server: " + err.Error())
		os.Exit(1)
	}

	router.GET("/tx/:txHash", getTxByHash)
	router.Run("localhost:8000")
}

func getTxByHash(c *gin.Context) {
	txHash := c.Param("txHash")

	domain := c.Query("domain")
	domainInt, err := strconv.ParseInt(domain, 10, 0)
	if domain != "" && err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "unable to parse domain"})
	}

	if tx, ok := State.Load(txHash); ok && domain == "" || (domain != "" && tx.Msgs[0].SourceDomain == types.Domain(domainInt)) {
		c.JSON(http.StatusOK, tx.Msgs)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"message": "message not found"})
}

func Parse(file string) (*types.Config, error) {
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

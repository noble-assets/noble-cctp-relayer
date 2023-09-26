package cmd

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
)

var (
	Cfg     config.Config
	cfgFile string
	verbose bool

	Logger log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "noble-cctp-relayer",
	Short: "A CLI tool for relaying CCTP messages",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
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

		Cfg = config.Parse(cfgFile)
		Logger.Info("successfully parsed config file", "location", cfgFile)
		// set defaults

		// if start block not set, default to latest
		if Cfg.Networks.Source.Ethereum.StartBlock == 0 {
			client, _ := ethclient.Dial(Cfg.Networks.Source.Ethereum.RPC)
			defer client.Close()
			header, _ := client.HeaderByNumber(context.Background(), nil)
			Cfg.Networks.Source.Ethereum.StartBlock = header.Number.Uint64()
		}

		// start api server
		go startApi()
	})
}

func startApi() {
	router := gin.Default()
	router.GET("/tx/:hash", getTxByHash)
	router.Run("localhost:8000")
}

func getTxByHash(c *gin.Context) {
	id := c.Param("hash")

	found := false
	if message, ok := State.Load(LookupKey("mint", id)); ok {
		c.IndentedJSON(http.StatusOK, message)
		found = true
	}
	if message, ok := State.Load(LookupKey("forward", id)); ok {
		c.IndentedJSON(http.StatusOK, message)
		found = true
	}
	if found {
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "message not found"})
}

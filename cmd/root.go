package cmd

import (
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
	Use:   "rly",
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

	if message, ok := State.Load(id); ok {
		c.IndentedJSON(http.StatusOK, message)
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "message not found"})
}

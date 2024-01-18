package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"

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

		Logger.Info(Cfg.Networks.Source.Ethereum.RPC)
		// Set minter addresses from priv keys
		for i, minter := range Cfg.Networks.Minters {
			switch i {
			case 0:
				_, address, err := ethereum.GetEcdsaKeyAddress(minter.MinterPrivateKey)
				if err != nil {
					Logger.Error(fmt.Sprintf("Unable to parse ecdsa key from source %d", i))
					os.Exit(1)
				}
				minter.MinterAddress = address
				Cfg.Networks.Minters[0] = minter
			case 4:
				keyBz, err := hex.DecodeString(minter.MinterPrivateKey)
				if err != nil {
					Logger.Error(fmt.Sprintf("Unable to parse key from source %d", i))
					os.Exit(1)
				}
				privKey := secp256k1.PrivKey{Key: keyBz}
				address, err := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
				if err != nil {
					Logger.Error(fmt.Sprintf("Unable to parse ecdsa key from source %d", i))
					os.Exit(1)
				}
				minter.MinterAddress = address
				Cfg.Networks.Minters[4] = minter
			}
		}

		// Set default listener blocks

		// if Ethereum start block not set, default to latest
		if Cfg.Networks.Source.Ethereum.Enabled && Cfg.Networks.Source.Ethereum.StartBlock == 0 {
			client, _ := ethclient.Dial(Cfg.Networks.Source.Ethereum.RPC)
			defer client.Close()
			header, _ := client.HeaderByNumber(context.Background(), nil)
			Cfg.Networks.Source.Ethereum.StartBlock = header.Number.Uint64()
		}

		// if Noble start block not set, default to latest
		if Cfg.Networks.Source.Noble.Enabled && Cfg.Networks.Source.Noble.StartBlock == 0 {
			rawResponse, _ := http.Get(Cfg.Networks.Source.Noble.RPC + "/block")
			body, _ := io.ReadAll(rawResponse.Body)
			response := types.BlockResponse{}
			_ = json.Unmarshal(body, &response)
			height, _ := strconv.ParseInt(response.Result.Block.Header.Height, 10, 0)
			Cfg.Networks.Source.Noble.StartBlock = uint64(height)
		}

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

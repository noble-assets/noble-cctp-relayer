package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

var (
	cfgFile string
	cfgYaml ConfigYaml
	conf    config.Config
	rootCmd = &cobra.Command{
		Use:   "cctp-relayer",
		Short: "A CLI tool for relaying CCTP messages to Noble",
		Long:  `A CLI tool for relaying Cross Chain Transfer Protocol messages to Noble.`,
	}
)

type ConfigYaml struct {
	Networks struct {
		Ethereum struct {
			RPC                string `yaml:"rpc"`
			TokenMessenger     string `yaml:"token-messenger"`
			MessageTransmitter string `yaml:"message-transmitter"`
		} `yaml:"ethereum"`
		Noble struct {
			RPC           string `yaml:"rpc"`
			DestinationId uint32 `yaml:"destination-id"`
		} `yaml:"noble"`
	} `yaml:"networks"`
	Indexer struct {
		StartBlock          int64    `yaml:"start-block"`
		AttestationBaseUrl  string   `yaml:"attestation-base-url"`
		ValidTokenAddresses []string `yaml:"valid-token-addresses"`
	} `yaml:"indexer"`
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// initConfig guarantees config struct will be set before all subcommands are executed
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

}

// read config file
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		log.Fatal("Configuration must be passed in manually")
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = yaml.Unmarshal(data, &cfgYaml)
	if err != nil {
		fmt.Println(err)
		return
	}

	// read env variables
	//viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	conf.Networks.Ethereum.RPC = cfgYaml.Networks.Ethereum.RPC
	conf.Networks.Ethereum.TokenMessenger = cfgYaml.Networks.Ethereum.TokenMessenger
	conf.Networks.Ethereum.MessageTransmitter = cfgYaml.Networks.Ethereum.MessageTransmitter
	conf.Networks.Noble.RPC = cfgYaml.Networks.Noble.RPC
	conf.Networks.Noble.DestinationId = cfgYaml.Networks.Noble.DestinationId
	conf.Indexer.StartBlock = cfgYaml.Indexer.StartBlock
	conf.Indexer.AttestationBaseUrl = cfgYaml.Indexer.AttestationBaseUrl
	conf.Indexer.ValidTokenAddresses = map[string]bool{}
	for _, address := range cfgYaml.Indexer.ValidTokenAddresses {
		conf.Indexer.ValidTokenAddresses[address] = true
	}

}

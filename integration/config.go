package integration_testing

import (
	"os"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"gopkg.in/yaml.v3"
)

var cfg *types.Config                      // app config
var integrationWallets *IntegrationWallets // for testing secrets

var nobleCfg *noble.ChainConfig
var ethCfg *ethereum.ChainConfig

var logger log.Logger
var err error

var nobleChain types.Chain
var ethChain types.Chain

// Sepolia
const TokenMessengerAddress = "0x9f3B8679c73C2Fef8b59B4f3444d4e156fb70AA5"
const UsdcAddress = "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"

var sequenceMap *types.SequenceMap

func setupTestIntegration() func() {
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))

	// cctp relayer app config setup for sepolia netowrk
	cfg, err = cmd.Parse("../.ignore/testnet.yaml")
	if err != nil {
		logger.Error("Error parsing relayer config")
		os.Exit(1)
	}

	// extra wallets to keep relayer wallet separate from test transaction
	// see config/sample-integration-config.yaml
	err = ParseIntegration("../.ignore/integration.yaml")
	if err != nil {
		logger.Error("Error parsing integration wallets")
		os.Exit(1)
	}

	nobleCfg = cfg.Chains["noble"].(*noble.ChainConfig)
	ethCfg = cfg.Chains["sepolia"].(*ethereum.ChainConfig)

	sequenceMap = types.NewSequenceMap()

	nobleChain, err = nobleCfg.Chain("noble")
	if err != nil {
		logger.Error("Error creating new chain", "err", err)
		os.Exit(1)
	}

	ethChain, err = ethCfg.Chain("eth")
	if err != nil {
		logger.Error("Error creating new chain", "err", err)
		os.Exit(1)
	}

	return func() {
		// teardown
	}
}

// Wallets used for integration testing
type IntegrationWallets struct {
	Networks struct {
		Ethereum struct {
			Address    string `yaml:"address"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"ethereum"`
		Noble struct {
			Address    string `yaml:"address"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"noble"`
	} `yaml:"networks"`
}

func ParseIntegration(file string) (err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &integrationWallets)
	if err != nil {
		return err
	}

	return nil
}

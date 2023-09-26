package integration_testing

import (
	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"os"

	"gopkg.in/yaml.v3"
)

var testCfg Config    // for testing secrets
var cfg config.Config // app config
var logger log.Logger

// goerli
const TokenMessengerAddress = "0xd0c3da58f55358142b8d3e06c1c30c5c6114efe8"
const TokenMessengerWithMetadataAddress = "0x1ae045d99236365cbdc1855acd2d2cfc232d04d1"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"

var sequenceMap *types.SequenceMap

func setupTest() func() {
	// setup
	testCfg = Parse("../.ignore/integration.yaml")
	cfg = config.Parse("../.ignore/testnet.yaml")
	logger = log.NewLogger(os.Stdout)

	_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
		cfg.Networks.Destination.Noble.API,
		cfg.Networks.Minters[4].MinterAddress)

	if err != nil {
		logger.Error("Error retrieving account sequence")
		os.Exit(1)
	}
	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(4), nextMinterSequence)

	return func() {
		// teardown
	}
}

type Config struct {
	Networks struct {
		Ethereum struct {
			RPC        string `yaml:"rpc"`
			PrivateKey string `yaml:"private_key"`
			FaucetUrl  string `yaml:"faucet_url"`
		} `yaml:"ethereum"`
		Noble struct {
			RPC       string `yaml:"rpc"`
			FaucetUrl string `yaml:"faucet_url"`
		} `yaml:"noble"`
	} `yaml:"networks"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)

	return
}

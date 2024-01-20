package integration_testing

import (
	"os"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"

	"gopkg.in/yaml.v3"
)

var testCfg *types.Config // for testing secrets
var cfg *types.Config     // app config
var logger log.Logger
var err error

// goerli
const TokenMessengerAddress = "0xd0c3da58f55358142b8d3e06c1c30c5c6114efe8"
const TokenMessengerWithMetadataAddress = "0x1ae045d99236365cbdc1855acd2d2cfc232d04d1"
const UsdcAddress = "0x07865c6e87b9f70255377e024ace6630c1eaa37f"

var sequenceMap *types.SequenceMap

func setupTest() func() {
	// setup
	testCfg, err = cmd.Parse("../.ignore/integration.yaml")
	cfg, err = cmd.Parse("../.ignore/testnet.yaml")
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.DebugLevel))

	_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
		cfg.Networks.Destination.Noble.API,
		cfg.Networks.Minters[4].MinterAddress)

	if err != nil {
		logger.Error("Error retrieving account sequence")
		os.Exit(1)
	}
	sequenceMap = types.NewSequenceMap()
	sequenceMap.Put(uint32(4), nextMinterSequence)

	for i, minter := range cfg.Networks.Minters {
		switch i {
		case 0:
			minter.MinterAddress = "0x971c54a6Eb782fAccD00bc3Ed5E934Cc5bD8e3Ef"
			cfg.Networks.Minters[0] = minter
		case 4:
			minter.MinterAddress = "noble1ar2gaqww6aphxd9qve5qglj8kqq96je6a4yrhj"
			cfg.Networks.Minters[4] = minter
		}
	}

	return func() {
		// teardown
	}
}

type Config struct {
	Networks struct {
		Ethereum struct {
			RPC        string `yaml:"rpc"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"ethereum"`
		Noble struct {
			RPC        string `yaml:"rpc"`
			PrivateKey string `yaml:"private_key"`
		} `yaml:"noble"`
	} `yaml:"networks"`
}

func Parse(file string) (cfg Config) {
	data, _ := os.ReadFile(file)
	_ = yaml.Unmarshal(data, &cfg)

	return
}

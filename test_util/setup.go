package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
)

var EnvFile = os.ExpandEnv("$GOPATH/src/github.com/strangelove-ventures/noble-cctp-relayer/.env")

func init() {
	err := godotenv.Load(EnvFile)
	if err != nil {
		fmt.Println(fmt.Errorf("error loading env file"))
		os.Exit(1)
	}
}

func ConfigSetup(t *testing.T) (a *cmd.AppState, registeredDomains map[types.Domain]types.Chain) {

	var testConfig = types.Config{
		Chains: map[string]types.ChainConfig{
			"noble": &noble.ChainConfig{
				ChainID: "grand-1",
				RPC:     os.Getenv("NOBLE_RPC"),
			},
			"ethereum": &ethereum.ChainConfig{
				ChainID:          11155111,
				Domain:           types.Domain(0),
				MinterPrivateKey: "1111111111111111111111111111111111111111111111111111111111111111",
				RPC:              os.Getenv("SEPOLIA_RPC"),
				WS:               os.Getenv("SEPOLIA_WS"),
			},
		},
		Circle: types.CircleSettings{
			AttestationBaseUrl: "https://iris-api-sandbox.circle.com/attestations/",
			FetchRetries:       0,
			FetchRetryInterval: 3,
		},

		EnabledRoutes: map[types.Domain][]types.Domain{
			0: {4},
			4: {0},
		},
	}

	a = cmd.NewAppState()
	a.LogLevel = "debug"
	a.InitLogger()
	a.Config = &testConfig

	registeredDomains = make(map[types.Domain]types.Chain)
	for name, cfgg := range a.Config.Chains {
		c, err := cfgg.Chain(name)
		require.NoError(t, err, "Error creating chain")

		registeredDomains[c.Domain()] = c
	}

	return a, registeredDomains

}

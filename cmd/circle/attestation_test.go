package circle_test

import (
	"cosmossdk.io/log"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var cfg config.Config
var logger log.Logger

func init() {
	cfg.Circle.AttestationBaseUrl = "https://iris-api-sandbox.circle.com/attestations/"
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
}

func TestAttestationIsReady(t *testing.T) {
	resp, found := circle.CheckAttestation(cfg, logger, "85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe")
	require.True(t, found)
	require.Equal(t, "complete", resp.Status)
}

func TestAttestationNotFound(t *testing.T) {
	resp, found := circle.CheckAttestation(cfg, logger, "not an attestation")
	require.False(t, found)
	require.Nil(t, resp)
}

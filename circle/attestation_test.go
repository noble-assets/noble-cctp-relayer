package circle_test

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var cfg types.Config
var logger log.Logger

func init() {
	cfg.Circle.AttestationBaseURL = "https://iris-api-sandbox.circle.com/attestations/"
	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
}

func TestAttestationIsReady(t *testing.T) {
	resp := circle.CheckAttestation(cfg.Circle.AttestationBaseURL, logger, "85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe", "", 0, 4)
	require.NotNil(t, resp)
	require.Equal(t, "complete", resp.Status)
}

func TestAttestationNotFound(t *testing.T) {
	resp := circle.CheckAttestation(cfg.Circle.AttestationBaseURL, logger, "not an attestation", "", 0, 4)
	require.Nil(t, resp)
}

func TestAttestationWithoutEndingSlash(t *testing.T) {
	startURL := cfg.Circle.AttestationBaseURL
	cfg.Circle.AttestationBaseURL = startURL[:len(startURL)-1]

	resp := circle.CheckAttestation(cfg.Circle.AttestationBaseURL, logger, "85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe", "", 0, 4)
	require.NotNil(t, resp)
	require.Equal(t, "complete", resp.Status)

	cfg.Circle.AttestationBaseURL = startURL
}

func TestAttestationWithLeading0x(t *testing.T) {
	resp := circle.CheckAttestation(cfg.Circle.AttestationBaseURL, logger, "0x85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe", "", 0, 4)
	require.NotNil(t, resp)
	require.Equal(t, "complete", resp.Status)
}

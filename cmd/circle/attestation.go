package circle

import (
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"io"
	"net/http"
	"time"
)

// CheckAttestation checks the iris api for attestation status and returns true if attestation is complete
func CheckAttestation(cfg config.Config, logger log.Logger, irisLookupId string) *types.AttestationResponse {
	logger.Debug(fmt.Sprintf("Checking attestation for %s%s%s", cfg.Circle.AttestationBaseUrl, "0x", irisLookupId))

	client := http.Client{Timeout: 2 * time.Second}

	rawResponse, err := client.Get(cfg.Circle.AttestationBaseUrl + "0x" + irisLookupId)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		logger.Debug("non 200 response received")
		return nil
	}
	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		logger.Debug("unable to parse message body")
		return nil
	}

	response := types.AttestationResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		logger.Debug("unable to unmarshal response")
		return nil
	}

	return &response
}

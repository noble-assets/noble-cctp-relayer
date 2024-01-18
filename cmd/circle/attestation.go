package circle

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// CheckAttestation checks the iris api for attestation status and returns true if attestation is complete
func CheckAttestation(cfg config.Config, logger log.Logger, irisLookupId string, txHash string, sourceDomain, destDomain types.Domain) *types.AttestationResponse {
	logger.Debug(fmt.Sprintf("Checking attestation for %s%s%s for source tx %s from %d to %d", cfg.Circle.AttestationBaseUrl, "0x", irisLookupId, txHash, sourceDomain, destDomain))

	client := http.Client{Timeout: 2 * time.Second}

	rawResponse, err := client.Get(cfg.Circle.AttestationBaseUrl + "0x" + irisLookupId)
	if err != nil {
		logger.Debug("error during request: " + err.Error())
		return nil
	}
	if rawResponse.StatusCode != http.StatusOK {
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
	logger.Info(fmt.Sprintf("Attestation found for %s%s%s", cfg.Circle.AttestationBaseUrl, "0x", irisLookupId))

	return &response
}

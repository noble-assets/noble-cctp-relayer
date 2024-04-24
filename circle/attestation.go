package circle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// CheckAttestation checks the iris api for attestation status and returns true if attestation is complete
func CheckAttestation(attestationURL string, logger log.Logger, irisLookupID string, txHash string, sourceDomain, destDomain types.Domain) *types.AttestationResponse {
	logger.Debug(fmt.Sprintf("Checking attestation for %s%s%s for source tx %s from %d to %d", attestationURL, "0x", irisLookupID, txHash, sourceDomain, destDomain))

	client := http.Client{Timeout: 2 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attestationURL+"0x"+irisLookupID, nil)
	if err != nil {
		logger.Debug("error creating request: " + err.Error())
		return nil
	}

	rawResponse, err := client.Do(req)
	if err != nil {
		logger.Debug("error during request: " + err.Error())
		return nil
	}
	defer rawResponse.Body.Close()
	if rawResponse.StatusCode != http.StatusOK {
		logger.Debug("non 200 response received from Circles attestation API")
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
	logger.Info(fmt.Sprintf("Attestation found for %s%s%s", attestationURL, "0x", irisLookupID))

	return &response
}

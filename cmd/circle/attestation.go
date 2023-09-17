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
func CheckAttestation(cfg config.Config, logger log.Logger, irisLookupId string) (*types.AttestationResponse, bool) {
	logger.Info(fmt.Sprintf("Checking attestation for %s%s%s", cfg.Circle.AttestationBaseUrl, "0x", irisLookupId))

	client := http.Client{
		Timeout: time.Duration(cfg.Circle.FetchRetryInterval) * time.Second,
	}

	for i := uint32(0); i < cfg.Circle.FetchRetries+1; i++ {
		rawResponse, err := client.Get(cfg.Circle.AttestationBaseUrl + "0x" + irisLookupId)
		if rawResponse.StatusCode != http.StatusOK || err != nil {
			logger.Debug("non 200 response received")
			time.Sleep(2 * time.Second)
			logger.Debug("retrying...")
			continue
		}
		body, err := io.ReadAll(rawResponse.Body)
		if err != nil {
			logger.Debug("unable to parse message body")
			return nil, false
		}

		response := types.AttestationResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			logger.Debug("unable to unmarshal response")
			return nil, false
		}
		return &response, true
	}
	return nil, false
}

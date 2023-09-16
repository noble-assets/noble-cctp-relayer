package circle

import (
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"io"
	"net/http"
)

// CheckAttestation checks the iris api for attestation status and returns true if attestation is complete
func CheckAttestation(cfg config.Config, logger log.Logger, irisLookupId string) (*types.AttestationResponse, bool) {
	logger.Info(fmt.Sprintf("CheckAttestation for %s%s%s", cfg.AttestationBaseUrl, "0x", irisLookupId))

	rawResponse, err := http.Get(cfg.AttestationBaseUrl + "0x" + irisLookupId)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		logger.Debug("non 200 response received")
		return nil, false
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

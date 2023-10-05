package noble

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

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
	// set up client

	logger.Info(fmt.Sprintf(
		"Starting Noble listener at block %d looking back %d blocks",
		cfg.Networks.Source.Noble.StartBlock,
		cfg.Networks.Source.Noble.LookbackPeriod))

	// TODO multithreading
	// constantly query for blocks
	currentBlock := cfg.Networks.Source.Noble.StartBlock
	for {
		rawResponse, err := http.Get(fmt.Sprintf("https://rpc.testnet.noble.strange.love/tx_search?query=\"tx.height=%d\"", currentBlock))
		if err != nil {
			logger.Debug(fmt.Sprintf("unable to query Noble block %d", currentBlock))
			time.Sleep(5 * time.Second)
		}
		if rawResponse.StatusCode != http.StatusOK {
			logger.Debug(fmt.Sprintf("non 200 response received for Noble block %d", currentBlock))
		}

		body, err := io.ReadAll(rawResponse.Body)
		if err != nil {
			logger.Debug(fmt.Sprintf("unable to parse Noble block %d", currentBlock))
			time.Sleep(5 * time.Second)
		}

		response := types.GetBlockResultsResponse{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			logger.Debug(fmt.Sprintf("unable to unmarshal Noble block %d", currentBlock))
			time.Sleep(5 * time.Second)
		}

		for _, tx := range response.Result.Txs {
			parsedMsg, err := types.NobleLogToMessageState(tx)
			if err != nil {
				logger.Error("unable to parse Noble log into MessageState, skipping")
				continue
			}
			processingQueue <- parsedMsg
		}
		currentBlock += 1
	}
}

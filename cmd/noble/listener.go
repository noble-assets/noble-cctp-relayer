package noble

import (
	"cosmossdk.io/log"
	"encoding/json"
	"fmt"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
	// set up client

	logger.Info(fmt.Sprintf(
		"Starting Noble listener at block %d looking back %d blocks",
		cfg.Networks.Source.Noble.StartBlock,
		cfg.Networks.Source.Noble.LookbackPeriod))

	var wg sync.WaitGroup
	wg.Add(1)

	// enqueue block heights
	currentBlock := cfg.Networks.Source.Noble.StartBlock
	chainTip := GetNobleChainTip(cfg)
	blockQueue := make(chan uint64, 1000000)
	for i := currentBlock; i <= chainTip; i++ {
		blockQueue <- i
	}
	currentBlock = chainTip

	go func() {
		for {
			chainTip = GetNobleChainTip(cfg)
			if chainTip > currentBlock {
				for i := currentBlock + 1; i <= chainTip; i++ {
					blockQueue <- i
				}
			}
			time.Sleep(6 * time.Second)
		}
	}()

	// constantly query for blocks
	for i := 0; i < int(cfg.Networks.Source.Noble.Workers); i++ {
		go func() {
			for {
				currentBlock := <-blockQueue
				logger.Debug(fmt.Sprintf("Querying Noble block %d", currentBlock))
				rawResponse, err := http.Get(fmt.Sprintf("https://rpc.testnet.noble.strange.love/tx_search?query=\"tx.height=%d\"", currentBlock))
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to query Noble block %d", currentBlock))
					continue
				}
				if rawResponse.StatusCode != http.StatusOK {
					logger.Debug(fmt.Sprintf("non 200 response received for Noble block %d", currentBlock))
					time.Sleep(5 * time.Second)
					blockQueue <- currentBlock
					continue
				}

				body, err := io.ReadAll(rawResponse.Body)
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to parse Noble block %d", currentBlock))
					continue
				}

				response := types.BlockResultsResponse{}
				err = json.Unmarshal(body, &response)
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to unmarshal Noble block %d", currentBlock))
					continue
				}

				for _, tx := range response.Result.Txs {
					parsedMsg, err := types.NobleLogToMessageState(tx)
					if err != nil {
						logger.Debug("unable to parse Noble log into MessageState, skipping")
						continue
					}
					processingQueue <- parsedMsg
				}
			}
		}()
	}

	wg.Wait()
}

func GetNobleChainTip(cfg config.Config) uint64 {
	rawResponse, _ := http.Get(cfg.Networks.Source.Noble.RPC + "/block")
	body, _ := io.ReadAll(rawResponse.Body)

	response := types.BlockResponse{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(err.Error())
	}
	res, _ := strconv.ParseInt(response.Result.Block.Header.Height, 10, 0)
	return uint64(res)
}

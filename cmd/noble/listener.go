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

	// history
	for currentBlock <= chainTip {
		fmt.Println(fmt.Sprintf("1 added block to queue: %d", currentBlock))
		blockQueue <- currentBlock
		currentBlock++
	}

	// listen for new blocks
	go func() {
		for {
			chainTip = GetNobleChainTip(cfg)
			if chainTip >= currentBlock {
				for i := currentBlock; i <= chainTip; i++ {
					fmt.Println(fmt.Sprintf("2 added block to queue: %d", i))
					blockQueue <- i
				}
				currentBlock = chainTip + 1
			}
			time.Sleep(6 * time.Second)
		}
	}()

	// constantly query for blocks
	for i := 0; i < int(cfg.Networks.Source.Noble.Workers); i++ {
		go func() {
			for {
				block := <-blockQueue
				logger.Debug(fmt.Sprintf("Querying Noble block %d", block))
				rawResponse, err := http.Get(fmt.Sprintf("https://rpc.testnet.noble.strange.love/tx_search?query=\"tx.height=%d\"", block))
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to query Noble block %d", block))
					continue
				}
				if rawResponse.StatusCode != http.StatusOK {
					logger.Debug(fmt.Sprintf("non 200 response received for Noble block %d", block))
					time.Sleep(5 * time.Second)
					blockQueue <- block
					continue
				}

				body, err := io.ReadAll(rawResponse.Body)
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to parse Noble block %d", block))
					continue
				}

				response := types.BlockResultsResponse{}
				err = json.Unmarshal(body, &response)
				if err != nil {
					logger.Debug(fmt.Sprintf("unable to unmarshal Noble block %d", block))
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

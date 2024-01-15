package noble

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
	// set up client

	logger.Info(fmt.Sprintf("Starting Noble listener at block %d looking back %d blocks",
		cfg.Networks.Source.Noble.StartBlock,
		cfg.Networks.Source.Noble.LookbackPeriod))

	var wg sync.WaitGroup
	wg.Add(1)

	// enqueue block heights
	currentBlock := cfg.Networks.Source.Noble.StartBlock
	lookback := cfg.Networks.Source.Noble.LookbackPeriod
	chainTip := GetNobleChainTip(cfg)
	blockQueue := make(chan uint64, 1000000)

	// history
	currentBlock = currentBlock - lookback
	for currentBlock <= chainTip {
		blockQueue <- currentBlock
		currentBlock++
	}

	// listen for new blocks
	go func() {
		for {
			chainTip = GetNobleChainTip(cfg)
			if chainTip >= currentBlock {
				for i := currentBlock; i <= chainTip; i++ {
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
				rawResponse, err := http.Get(fmt.Sprintf("%s/tx_search?query=\"tx.height=%d\"", cfg.Networks.Source.Noble.RPC, block))
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
					parsedMsgs, err := types.NobleLogToMessageState(tx)
					if err != nil {
						continue
					}
					for _, parsedMsg := range parsedMsgs {
						logger.Info(fmt.Sprintf("New stream msg from %d with tx hash %s", parsedMsg.SourceDomain, parsedMsg.SourceTxHash))
						processingQueue <- parsedMsg
					}
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

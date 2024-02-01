package noble

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

func (n *Noble) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
) {
	logger = logger.With("chain", n.Name(), "chain_id", n.chainID, "domain", n.Domain())

	if n.startBlock == 0 {
		// get the latest block
		chainTip, err := n.chainTip(ctx)
		if err != nil {
			panic(fmt.Errorf("unable to get chain tip for noble: %w", err))
		}
		n.startBlock = chainTip
	}

	logger.Info(fmt.Sprintf("Starting Noble listener at block %d looking back %d blocks",
		n.startBlock,
		n.lookbackPeriod))

	accountNumber, _, err := n.AccountInfo(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to get account info for noble: %w", err))
	}

	n.accountNumber = accountNumber

	// enqueue block heights
	currentBlock := n.startBlock
	lookback := n.lookbackPeriod
	chainTip, err := n.chainTip(ctx)
	blockQueue := make(chan uint64, 1000000)

	// history
	currentBlock = currentBlock - lookback
	for currentBlock <= chainTip {
		blockQueue <- currentBlock
		currentBlock++
	}

	// listen for new blocks
	go func() {
		first := make(chan struct{}, 1)
		first <- struct{}{}
		for {
			timer := time.NewTimer(6 * time.Second)
			select {
			case <-first:
				timer.Stop()
				chainTip, err = n.chainTip(ctx)
				if err == nil {
					if chainTip >= currentBlock {
						for i := currentBlock; i <= chainTip; i++ {
							blockQueue <- i
						}
						currentBlock = chainTip + 1
					}
				}
			case <-timer.C:
				chainTip, err = n.chainTip(ctx)
				if err == nil {
					if chainTip >= currentBlock {
						for i := currentBlock; i <= chainTip; i++ {
							blockQueue <- i
						}
						currentBlock = chainTip + 1
					}
				}
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()

	// constantly query for blocks
	for i := 0; i < int(n.workers); i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					block := <-blockQueue
					res, err := n.cc.RPCClient.TxSearch(ctx, fmt.Sprintf("tx.height=%d", block), false, nil, nil, "")
					if err != nil || res == nil {
						logger.Debug(fmt.Sprintf("unable to query Noble block %d", block), "error:", err)
						blockQueue <- block
						continue
					}

					for _, tx := range res.Txs {
						parsedMsgs, err := txToMessageState(tx)
						if err != nil {
							logger.Error("unable to parse Noble log to message state", "err", err.Error())
							continue
						}
						for _, parsedMsg := range parsedMsgs {
							logger.Info(fmt.Sprintf("New stream msg with nonce %d from %d with tx hash %s", parsedMsg.Nonce, parsedMsg.SourceDomain, parsedMsg.SourceTxHash))
						}
						processingQueue <- &types.TxState{TxHash: tx.Hash.String(), Msgs: parsedMsgs}
					}
				}
			}
		}()
	}

	<-ctx.Done()
}

func (n *Noble) chainTip(ctx context.Context) (uint64, error) {
	res, err := n.cc.RPCClient.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to query status for noble: %w", err)
	}
	return uint64(res.SyncInfo.LatestBlockHeight), nil
}

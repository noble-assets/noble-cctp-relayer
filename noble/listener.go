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
		n.startBlock = n.latestBlock
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
	chainTip := n.latestBlock

	if n.blockQueueChannelSize == 0 {
		n.blockQueueChannelSize = defaultBlockQueueChannelSize
	}
	blockQueue := make(chan uint64, n.blockQueueChannelSize)

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
				chainTip = n.latestBlock
				if chainTip >= currentBlock {
					for i := currentBlock; i <= chainTip; i++ {
						blockQueue <- i
					}
					currentBlock = chainTip + 1
				}
			case <-timer.C:
				chainTip = n.latestBlock
				if chainTip >= currentBlock {
					for i := currentBlock; i <= chainTip; i++ {
						blockQueue <- i
					}
					currentBlock = chainTip + 1
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

	go n.flushMechanism(ctx, logger, blockQueue)

	<-ctx.Done()
}

func (n *Noble) flushMechanism(
	ctx context.Context,
	logger log.Logger,
	blockQueue chan uint64,
) {
	for {
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
			latestBlock := n.latestBlock

			if n.lastFlushedBlock == 0 {
				n.lastFlushedBlock = latestBlock
			}
			lastFlushedBlock := n.lastFlushedBlock

			flushStart := lastFlushedBlock - n.lookbackPeriod

			logger.Info(fmt.Sprintf("flushing... start-block: %d end-block: %d", flushStart, latestBlock))

			for i := flushStart; i <= latestBlock; i++ {
				blockQueue <- i
			}
			n.lastFlushedBlock = latestBlock

			logger.Info("flush complete")

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (n *Noble) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, loop time.Duration) {
	logger.With("routine", "TrackLatestBlockHeight", "chain", n.Name(), "domain", n.Domain())

	// first time
	res, err := n.cc.RPCClient.Status(ctx)
	if err != nil {
		logger.Error("unable to query Nobles latest height", "err", err)
	}
	n.latestBlock = uint64(res.SyncInfo.LatestBlockHeight)

	// then start loop on a timer
	for {
		timer := time.NewTimer(loop)
		select {
		case <-timer.C:
			res, err := n.cc.RPCClient.Status(ctx)
			if err != nil {
				logger.Error("unable to query Nobles latest height", "err", err)
				continue
			}
			n.latestBlock = uint64(res.SyncInfo.LatestBlockHeight)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

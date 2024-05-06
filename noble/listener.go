package noble

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var flushInterval time.Duration

func (n *Noble) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	flushOnlyMode bool,
	flushInterval_ time.Duration,
) {
	logger = logger.With("chain", n.Name(), "chain_id", n.chainID, "domain", n.Domain())

	flushInterval = flushInterval_

	if n.startBlock == 0 {
		n.startBlock = n.LatestBlock()
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
	chainTip := n.LatestBlock()

	if n.blockQueueChannelSize == 0 {
		n.blockQueueChannelSize = defaultBlockQueueChannelSize
	}
	blockQueue := make(chan uint64, n.blockQueueChannelSize)

	if !flushOnlyMode {
		// history
		currentBlock -= lookback
		for currentBlock <= chainTip {
			blockQueue <- currentBlock
			currentBlock++
		}

		// listen for new blocks
		go func() {
			// inner function to queue blocks
			queueBlocks := func() {
				chainTip = n.LatestBlock()
				if chainTip >= currentBlock {
					for i := currentBlock; i <= chainTip; i++ {
						blockQueue <- i
					}
					currentBlock = chainTip + 1
				}
			}

			// initial queue
			queueBlocks()

			for {
				timer := time.NewTimer(6 * time.Second)
				select {
				case <-timer.C:
					queueBlocks()
				case <-ctx.Done():
					timer.Stop()
					return
				}
			}
		}()
	}

	// constantly query for blocks
	for i := 0; i < int(n.workers); i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case block := <-blockQueue:
					res, err := n.cc.RPCClient.TxSearch(ctx, fmt.Sprintf("tx.height=%d", block), false, nil, nil, "")
					if err != nil || res == nil {
						logger.Debug(fmt.Sprintf("Unable to query Noble block %d. Will retry.", block), "error:", err)
						blockQueue <- block
						continue
					}

					for _, tx := range res.Txs {
						parsedMsgs, err := txToMessageState(tx)
						if err != nil {
							logger.Error("Unable to parse Noble log to message state", "err", err.Error())
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

	if flushInterval > 0 {
		go n.flushMechanism(ctx, logger, blockQueue, flushOnlyMode)
	}

	<-ctx.Done()
}

// flushMechanism looks back over the chain history every specified flushInterval.
//
// Each chain is configured with a lookback period which signifies how many blocks to look back
// at each interval. The flush mechanism will start from the last flushed block and will rescan
// the lookback period and consume all messages in that range. The flush mechanism will not flush
// all the way to the chain's latest block to avoid consuming messages that are still in the queue.
// There will be a minimum gap of the lookback period between the last flushed block and the latest block.
//
// Note: The first time the flush mechanism is run, it will set the lastFlushedBlock to the latest block
// minus twice the lookback period.
func (n *Noble) flushMechanism(
	ctx context.Context,
	logger log.Logger,
	blockQueue chan uint64,
	flushOnlyMode bool,
) {
	logger.Info(fmt.Sprintf("Starting flush mechanism. Will flush every %v", flushInterval))

	// extraFlushBlocks is used to add an extra space between latest height and last flushed block
	// this setting should only be used for the secondary, flush only relayer
	extraFlushBlocks := uint64(0)
	if flushOnlyMode {
		extraFlushBlocks = 2 * n.lookbackPeriod
	}

	for {
		timer := time.NewTimer(flushInterval)
		select {
		case <-timer.C:
			latestBlock := n.LatestBlock()

			// test to see that the rpc is available before attempting flush
			res, err := n.cc.RPCClient.Status(ctx)
			if err != nil {
				logger.Error(fmt.Sprintf("Skipping flush... error reaching out to rpc, will retry flush in %v", flushInterval))
				continue
			}
			if res.SyncInfo.CatchingUp {
				logger.Error(fmt.Sprintf("Skipping flush... rpc still catching, will retry flush in %v", flushInterval))
				continue
			}

			// initialize first lastFlushedBlock if not set
			if n.lastFlushedBlock == 0 {
				n.lastFlushedBlock = latestBlock - (2*n.lookbackPeriod + extraFlushBlocks)

				if latestBlock < n.lookbackPeriod {
					n.lastFlushedBlock = 0
				}
			}

			// start from the last block it flushed
			startBlock := n.lastFlushedBlock

			// set finish block to be latestBlock - lookbackPeriod
			finishBlock := latestBlock - (n.lookbackPeriod + extraFlushBlocks)

			if startBlock >= finishBlock {
				logger.Debug("No new blocks to flush")
				continue
			}

			logger.Info(fmt.Sprintf("Flush started from %d to %d (current height: %d, lookback period: %d)", startBlock, finishBlock, latestBlock, n.lookbackPeriod))

			for i := startBlock; i <= finishBlock; i++ {
				blockQueue <- i
			}
			n.lastFlushedBlock = finishBlock

			logger.Info("Flush complete")

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (n *Noble) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, m *relayer.PromMetrics) {
	logger.With("routine", "TrackLatestBlockHeight", "chain", n.Name(), "domain", n.Domain())

	d := fmt.Sprint(n.Domain())

	// inner function to update block height
	updateBlockHeight := func() {
		res, err := n.cc.RPCClient.Status(ctx)
		if err != nil {
			logger.Error("Unable to query Nobles latest height", "err", err)
		} else {
			n.SetLatestBlock(uint64(res.SyncInfo.LatestBlockHeight))
			if m != nil {
				m.SetLatestHeight(n.Name(), d, res.SyncInfo.LatestBlockHeight)
			}
		}
	}

	// initial call
	updateBlockHeight()

	// then start loop on a timer
	for {
		timer := time.NewTimer(6 * time.Second)
		select {
		case <-timer.C:
			updateBlockHeight()
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (n *Noble) WalletBalanceMetric(ctx context.Context, logger log.Logger, m *relayer.PromMetrics) {
	// Relaying is free. No need to track noble balance.
}

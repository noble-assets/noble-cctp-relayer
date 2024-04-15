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

	// history
	currentBlock = currentBlock - lookback
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
		go n.flushMechanism(ctx, logger, blockQueue)
	}

	<-ctx.Done()
}

func (n *Noble) flushMechanism(
	ctx context.Context,
	logger log.Logger,
	blockQueue chan uint64,
) {

	logger.Debug(fmt.Sprintf("Flush mechanism started. Will flush every %v", flushInterval))

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

			if n.lastFlushedBlock == 0 {
				n.lastFlushedBlock = latestBlock
			}
			lastFlushedBlock := n.lastFlushedBlock

			flushStart := lastFlushedBlock - n.lookbackPeriod

			logger.Info(fmt.Sprintf("Flush started from: %d to: %d", flushStart, latestBlock))

			for i := flushStart; i <= latestBlock; i++ {
				blockQueue <- i
			}
			n.lastFlushedBlock = latestBlock

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

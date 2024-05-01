package ethereum

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pascaldekloe/etherstream"

	"cosmossdk.io/log"

	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// errSignal allows broadcasting an error value to multiple receivers.
type errSignal struct {
	Ready chan struct{}
}

// StartListener starts the ethereum websocket subscription, queries history pertaining to the lookback period,
// and starts the reoccurring flush
//
// If an error occurs in websocket stream, this function will handle relevant sub routines and then re-run itself.
func (e *Ethereum) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	flushInterval time.Duration,
) {
	logger = logger.With("chain", e.name, "chain_id", e.chainID, "domain", e.domain)

	messageTransmitter, err := content.ReadFile("abi/MessageTransmitter.json")
	if err != nil {
		logger.Error("Unable to read MessageTransmitter abi", "err", err)
		os.Exit(1)
	}
	messageTransmitterABI, err := abi.JSON(bytes.NewReader(messageTransmitter))
	if err != nil {
		logger.Error("Unable to parse MessageTransmitter abi", "err", err)
		os.Exit(1)
	}

	messageSent := messageTransmitterABI.Events["MessageSent"]
	messageTransmitterAddress := common.HexToAddress(e.messageTransmitterAddress)

	sig := &errSignal{
		Ready: make(chan struct{}),
	}

	// start main stream (does not account for lookback period or specific start block)
	stream, sub, history := e.startMainStream(ctx, logger, messageSent, messageTransmitterAddress)

	go e.consumeStream(ctx, logger, processingQueue, messageSent, messageTransmitterABI, stream, sig)
	consumeHistory(logger, history, processingQueue, messageSent, messageTransmitterABI)

	// get history from (start block - lookback) up until latest block
	latestBlock := e.LatestBlock()
	start := latestBlock
	if e.startBlock != 0 {
		start = e.startBlock
	}
	startLookback := start - e.lookbackPeriod
	logger.Info(fmt.Sprintf("Getting history from %d: starting at: %d looking back %d blocks", startLookback, start, e.lookbackPeriod))
	e.getAndConsumeHistory(ctx, logger, processingQueue, messageSent, messageTransmitterAddress, messageTransmitterABI, startLookback, latestBlock)

	logger.Info("Finished getting history")

	if flushInterval > 0 {
		go e.flushMechanism(ctx, logger, processingQueue, messageSent, messageTransmitterAddress, messageTransmitterABI, flushInterval, sig)
	}

	// listen for errors in the main websocket stream
	// if error occurs, trigger sig.Ready
	// This will cancel `consumeStream` and `flushMechanism` routines
	select {
	case <-ctx.Done():
		return
	case err := <-sub.Err():
		logger.Error("Websocket disconnected. Reconnecting...", "err", err)
		close(sig.Ready)

		// restart
		e.startBlock = e.lastFlushedBlock
		time.Sleep(10 * time.Millisecond)
		e.StartListener(ctx, logger, processingQueue, flushInterval)
		return
	}
}

func (e *Ethereum) startMainStream(
	ctx context.Context,
	logger log.Logger,
	messageSent abi.Event,
	messageTransmitterAddress common.Address,
) (stream <-chan ethtypes.Log, sub ethereum.Subscription, history []ethtypes.Log) {
	var err error

	etherReader := etherstream.Reader{Backend: e.wsClient}

	latestBlock := e.LatestBlock()

	// start initial stream (start-block and lookback period handled separately)
	logger.Info("Starting Ethereum listener")

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{messageSent.ID}},
		FromBlock: big.NewInt(int64(latestBlock)),
	}

	queryAttempt := 1
	for {
		// websockets do not query history
		// https://github.com/ethereum/go-ethereum/issues/15063
		stream, sub, history, err = etherReader.QueryWithHistory(ctx, &query)
		if err != nil {
			logger.Error("Unable to subscribe to logs", "attempt", queryAttempt, "err", err)
			queryAttempt++
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	return stream, sub, history
}

func (e *Ethereum) getAndConsumeHistory(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	messageSent abi.Event,
	messageTransmitterAddress common.Address,
	messageTransmitterABI abi.ABI,
	start, end uint64) {
	var toUnSub ethereum.Subscription
	var history []ethtypes.Log
	var err error

	if start > end {
		logger.Error(fmt.Sprintf("Unable to get history from %d to %d where the start block is greater than the end block", start, end))
		return
	}

	// handle historical queries in chunks (some websockets only allow small history queries)
	const chunkSize = uint64(100)
	chunk := 1
	totalChunksNeeded := (end - start + chunkSize - 1) / chunkSize

	for start < end {
		fromBlock := start
		toBlock := start + chunkSize
		if toBlock > end {
			toBlock = end
		}

		logger.Debug(fmt.Sprintf("Looking back in chunks of %d: chunk: %d/%d start-block: %d end-block: %d", chunkSize, chunk, totalChunksNeeded, fromBlock, toBlock))

		etherReader := etherstream.Reader{Backend: e.wsClient}

		query := ethereum.FilterQuery{
			Addresses: []common.Address{messageTransmitterAddress},
			Topics:    [][]common.Hash{{messageSent.ID}},
			FromBlock: big.NewInt(int64(fromBlock)),
			ToBlock:   big.NewInt(int64(toBlock)),
		}
		queryAttempt := 1
		for {
			_, toUnSub, history, err = etherReader.QueryWithHistory(ctx, &query)
			if err != nil {
				// TODO: add metrics for this log
				logger.Error(fmt.Sprintf("Unable to query history from %d to %d. attempt: %d", start, end, queryAttempt), "err", err)
				queryAttempt++
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		toUnSub.Unsubscribe()
		consumeHistory(logger, history, processingQueue, messageSent, messageTransmitterABI)

		start += chunkSize
		chunk++
	}
}

// consumeHistory consumes the history from a QueryWithHistory() go-ethereum call.
// it passes messages to the processingQueue
func consumeHistory(
	logger log.Logger,
	history []ethtypes.Log,
	processingQueue chan *types.TxState,
	messageSent abi.Event,
	messageTransmitterABI abi.ABI,
) {
	for i := range history {
		historicalLog := history[i]
		parsedMsg, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &historicalLog)
		if err != nil {
			logger.Error("Unable to parse history log into MessageState, skipping", "tx hash", historicalLog.TxHash.Hex(), "err", err)
			continue
		}
		logger.Info(fmt.Sprintf("New historical msg from source domain %d with tx hash %s", parsedMsg.SourceDomain, parsedMsg.SourceTxHash))

		processingQueue <- &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}
	}
}

// consumeStream consumes incoming transactions from a QueryWithHistory() go-ethereum call.
// if the websocket is disconnect, it restarts the stream using the last seen block height as the start height.
func (e *Ethereum) consumeStream(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	messageSent abi.Event,
	messageTransmitterABI abi.ABI,
	stream <-chan ethtypes.Log,
	sig *errSignal,

) {
	logger.Info("Starting consumption of incoming stream")
	var txState *types.TxState
	for {
		select {
		case <-ctx.Done():
			return
		case <-sig.Ready:
			logger.Debug("Websocket disconnected... Stopped consuming stream. Will restart after websocket is re-established")
			return
		case streamLog := <-stream:
			parsedMsg, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &streamLog)
			if err != nil {
				logger.Error("Unable to parse ws log into MessageState, skipping", "source tx", streamLog.TxHash.Hex(), "err", err)
				continue
			}
			logger.Info(fmt.Sprintf("New stream msg from %d with tx hash %s", parsedMsg.SourceDomain, parsedMsg.SourceTxHash))

			switch {
			case txState == nil:
				txState = &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}
			case parsedMsg.SourceTxHash != txState.TxHash:
				processingQueue <- txState
				txState = &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}
			default:
				txState.Msgs = append(txState.Msgs, parsedMsg)
			}
		default:
			if txState != nil {
				processingQueue <- txState
				txState = nil
			}
		}
	}
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
func (e *Ethereum) flushMechanism(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	messageSent abi.Event,
	messageTransmitterAddress common.Address,
	messageTransmitterABI abi.ABI,
	flushInterval time.Duration,
	sig *errSignal,
) {
	logger.Info(fmt.Sprintf("Starting flush mechanism. Will flush every %v", flushInterval))

	for {
		timer := time.NewTimer(flushInterval)
		select {
		case <-timer.C:
			latestBlock := e.LatestBlock()

			// initialize first lastFlushedBlock if not set
			if e.lastFlushedBlock == 0 {
				e.lastFlushedBlock = latestBlock - 2*e.lookbackPeriod

				if latestBlock < e.lookbackPeriod {
					e.lastFlushedBlock = 0
				}
			}

			// start from the last block it flushed
			startBlock := e.lastFlushedBlock

			// set finish block to be latestBlock - lookbackPeriod
			finishBlock := latestBlock - e.lookbackPeriod

			if startBlock >= finishBlock {
				logger.Debug("No new blocks to flush")
				continue
			}

			logger.Info(fmt.Sprintf("Flush started from %d to %d (current height: %d, lookback period: %d)", startBlock, finishBlock, latestBlock, e.lookbackPeriod))

			// consume from lastFlushedBlock to the finishBlock
			e.getAndConsumeHistory(ctx, logger, processingQueue, messageSent, messageTransmitterAddress, messageTransmitterABI, startBlock, finishBlock)

			// update lastFlushedBlock to the last block it flushed
			e.lastFlushedBlock = finishBlock

			logger.Info("Flush complete")

		// if main websocket stream is disconnected, stop flush. It will be restarted once websocket is reconnected
		case <-sig.Ready:
			timer.Stop()
			logger.Debug("Websocket disconnected... Flush stopped. Will restart after websocket is re-established")
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (e *Ethereum) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, m *relayer.PromMetrics) {
	logger.With("routine", "TrackLatestBlockHeight", "chain", e.name, "domain", e.domain)

	d := fmt.Sprint(e.domain)

	// helper function to query latest height and set metric
	queryHeightAndSetMetric := func() {
		// first time
		res, err := e.rpcClient.BlockNumber(ctx)
		if err != nil {
			logger.Error("Unable to query latest height", "err", err)
		} else {
			e.SetLatestBlock(res)
			if m != nil {
				m.SetLatestHeight(e.name, d, int64(res))
			}
		}
	}

	// initial query
	queryHeightAndSetMetric()

	// then start loop on a timer
	for {
		timer := time.NewTimer(30 * time.Second)
		select {
		case <-timer.C:
			queryHeightAndSetMetric()
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (e *Ethereum) WalletBalanceMetric(ctx context.Context, logger log.Logger, m *relayer.PromMetrics) {
	logger = logger.With("metric", "wallet balance", "chain", e.name, "domain", e.domain)
	queryRate := 5 * time.Minute

	account := common.HexToAddress(e.minterAddress)

	exponent := big.NewInt(int64(e.MetricsExponent))                                      // ex: 18
	scaleFactor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), exponent, nil)) // ex: 10^18

	// helper function to query balance and set metric
	queryBalanceAndSetMetric := func() {
		balance, err := e.rpcClient.BalanceAt(ctx, account, nil)
		if err != nil {
			logger.Error(fmt.Sprintf("Error querying balance. Will try again in %.2f sec", queryRate.Seconds()), "error", err)
		} else {
			balanceBigFloat := new(big.Float).SetInt(balance)
			balanceScaled, _ := new(big.Float).Quo(balanceBigFloat, scaleFactor).Float64()

			if m != nil {
				m.SetWalletBalance(e.name, e.minterAddress, e.MetricsDenom, balanceScaled)
			}
		}
	}

	// initial query
	queryBalanceAndSetMetric()

	for {
		timer := time.NewTimer(queryRate)
		select {
		case <-timer.C:
			queryBalanceAndSetMetric()
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

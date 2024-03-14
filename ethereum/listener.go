package ethereum

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"cosmossdk.io/log"
	ethereum "github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pascaldekloe/etherstream"
	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var (
	messageTransmitterABI     abi.ABI
	messageSent               abi.Event
	messageTransmitterAddress common.Address
)

func (e *Ethereum) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
) {
	logger = logger.With("chain", e.name, "chain_id", e.chainID, "domain", e.domain)

	messageTransmitter, err := content.ReadFile("abi/MessageTransmitter.json")
	if err != nil {
		logger.Error("unable to read MessageTransmitter abi", "err", err)
		os.Exit(1)
	}
	messageTransmitterABI, err = abi.JSON(bytes.NewReader(messageTransmitter))
	if err != nil {
		logger.Error("unable to parse MessageTransmitter abi", "err", err)
		os.Exit(1)
	}

	messageSent = messageTransmitterABI.Events["MessageSent"]
	messageTransmitterAddress = common.HexToAddress(e.messageTransmitterAddress)

	e.startListenerRoutines(ctx, logger, processingQueue)

}

// startListenerRoutines starts the ethereum websocket subscription, queries history pertaining to the lookback period,
// and starts the reoccurring flush
//
// we pass the subscription from the initial queryEth() to flushMechanism() so in the case the
// websocket becomes disconnect, we stop and then re-start the flush.
func (e *Ethereum) startListenerRoutines(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
) {

	// start main stream (does not account for lookback period)
	stream, sub, history := e.startMainStream(ctx, logger)
	go e.consumeStream(ctx, logger, processingQueue, stream, sub)
	consumeHistroy(logger, history, processingQueue)

	// query history pertaining to lookback period
	if e.lookbackPeriod != 0 {
		latestBlock := e.latestBlock
		start := latestBlock - e.lookbackPeriod
		end := latestBlock
		logger.Info(fmt.Sprintf("starting lookback of %d blocks", e.lookbackPeriod))
		e.getAndConsumeHistory(ctx, logger, processingQueue, start, end)
		logger.Info(fmt.Sprintf("finished lookback of %d blocks", e.lookbackPeriod))
	}

	go e.flushMechanism(ctx, logger, processingQueue, sub)

}

func (e *Ethereum) startMainStream(
	ctx context.Context,
	logger log.Logger,
) (stream <-chan ethtypes.Log, sub ethereum.Subscription, history []ethtypes.Log) {

	var err error

	etherReader := etherstream.Reader{Backend: e.wsClient}

	if e.startBlock == 0 {
		e.startBlock = e.latestBlock
	}

	latestBlock := e.latestBlock

	// start initial stream (lookback period handled separately)
	logger.Info(fmt.Sprintf("Starting Ethereum listener at block %d", e.startBlock))

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
			logger.Error("unable to subscribe to logs", "attempt", queryAttempt, "err", err)
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
	start, end uint64) {

	var toUnSub ethereum.Subscription
	var history []ethtypes.Log
	var err error

	// handle historical queries in chunks (some websockets only allow small history queries)
	chunkSize := uint64(100)
	chunk := 1
	totalChunksNeeded := (end - start) / chunkSize
	if (end-start)%chunkSize > 0 || totalChunksNeeded == 0 {
		totalChunksNeeded++
	}

	for start < end {
		fromBlock := start
		toBlock := start + chunkSize
		if toBlock > end {
			toBlock = end
		}

		logger.Debug(fmt.Sprintf("looking back in chunks of %d: chunk: %d/%d start-block: %d end-block: %d", chunkSize, chunk, totalChunksNeeded, fromBlock, toBlock))

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
				logger.Error("unable to query history from %d to %d. attempt: %d", start, end, queryAttempt)
				queryAttempt++
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		toUnSub.Unsubscribe()
		consumeHistroy(logger, history, processingQueue)

		start += chunkSize
		chunk++
	}
}

// consumeHistroy consumes the hisroty from a QueryWithHistory() go-ethereum call.
// it passes messages to the processingQueue
func consumeHistroy(
	logger log.Logger,
	history []ethtypes.Log,
	processingQueue chan *types.TxState,
) {
	for _, historicalLog := range history {
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
	stream <-chan ethtypes.Log,
	sub ethereum.Subscription,
) {
	var txState *types.TxState
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-sub.Err():
			// setting start block to 0 will start listener from latsest height.
			// in the rare case we are waiting for the websocket to come back on line for a long period of time,
			// we'll rely on the latestFlushBlock to flush out all missted transactions
			e.startBlock = 0
			logger.Error("connection closed. Restarting...", "err", err)
			e.startListenerRoutines(ctx, logger, processingQueue)
			return
		case streamLog := <-stream:
			parsedMsg, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &streamLog)
			if err != nil {
				logger.Error("Unable to parse ws log into MessageState, skipping", "source tx", streamLog.TxHash.Hex(), "err", err)
				continue
			}
			logger.Info(fmt.Sprintf("New stream msg from %d with tx hash %s", parsedMsg.SourceDomain, parsedMsg.SourceTxHash))
			if txState == nil {
				txState = &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}
			} else if parsedMsg.SourceTxHash != txState.TxHash {
				processingQueue <- txState
				txState = &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}
			} else {
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

func (e *Ethereum) flushMechanism(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	sub ethereum.Subscription,
) {

	for {
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
			latestBlock := e.latestBlock

			if e.lastFlushedBlock == 0 {
				e.lastFlushedBlock = latestBlock
			}

			start := e.lastFlushedBlock - e.lookbackPeriod

			logger.Info(fmt.Sprintf("flush started from %d to %d", start, latestBlock))

			e.getAndConsumeHistory(ctx, logger, processingQueue, start, latestBlock)

			logger.Info("flush complete")

		// if main websocket stream is disconnected, stop flush. It will be restarted once websocket is reconnected
		case <-sub.Err():
			timer.Stop()
			logger.Info("websocket disconnected, stopping flush mechanism. Will restart after websocket is re-established")
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (e *Ethereum) TrackLatestBlockHeight(ctx context.Context, logger log.Logger, loop time.Duration) {
	logger.With("routine", "TrackLatestBlockHeight", "chain", e.name, "domain", e.domain)

	// first time
	header, err := e.rpcClient.HeaderByNumber(ctx, nil)
	if err != nil {
		logger.Error("Error getting lastest block height:", err)
	}
	e.latestBlock = header.Number.Uint64()

	// then start loop on a timer
	for {
		timer := time.NewTimer(loop)
		select {
		case <-timer.C:
			header, err := e.rpcClient.HeaderByNumber(ctx, nil)
			if err != nil {
				logger.Error("Error getting lastest block height:", err)
				continue
			}
			e.latestBlock = header.Number.Uint64()

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (e *Ethereum) WalletBalanceMetric(ctx context.Context, logger log.Logger, m *relayer.PromMetrics) {
	logger = logger.With("metric", "wallet blance", "chain", e.name, "domain", e.domain)
	queryRate := 30 * time.Second

	var err error
	var client *ethclient.Client

	account := common.HexToAddress(e.minterAddress)

	exponent := big.NewInt(int64(e.MetricsExponent))                                      // ex: 18
	scaleFactor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), exponent, nil)) // ex: 10^18

	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	first := make(chan struct{}, 1)
	first <- struct{}{}
	createClient := true
	for {
		timer := time.NewTimer(queryRate)
		select {
		// don't wait the "queryRate" amount of time if this is the first time running
		case <-first:
			timer.Stop()
			if createClient {
				client, err = ethclient.DialContext(ctx, e.rpcURL)
				if err != nil {
					logger.Error(fmt.Sprintf("error dialing eth client. Will try again in %d sec", queryRate), "error", err)
					createClient = true
					continue
				}
			}
			balance, err := client.BalanceAt(ctx, account, nil)
			if err != nil {
				logger.Error(fmt.Sprintf("error querying balance. Will try again in %d sec", queryRate), "error", err)
				createClient = true
				continue
			}

			balanceBigFloat := new(big.Float).SetInt(balance)
			balanceScaled, _ := new(big.Float).Quo(balanceBigFloat, scaleFactor).Float64()

			m.SetWalletBalance(e.name, e.minterAddress, e.MetricsDenom, balanceScaled)

			createClient = false
		case <-timer.C:
			if createClient {
				client, err = ethclient.DialContext(ctx, e.rpcURL)
				if err != nil {
					logger.Error(fmt.Sprintf("error dialing eth client. Will try again in %d sec", queryRate), "error", err)
					createClient = true
					continue
				}
			}
			balance, err := client.BalanceAt(ctx, account, nil)
			if err != nil {
				logger.Error(fmt.Sprintf("error querying balance. Will try again in %d sec", queryRate), "error", err)
				createClient = true
				continue
			}

			balanceBigFloat := new(big.Float).SetInt(balance)
			balanceScaled, _ := new(big.Float).Quo(balanceBigFloat, scaleFactor).Float64()

			m.SetWalletBalance(e.name, e.minterAddress, e.MetricsDenom, balanceScaled)

			createClient = false
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

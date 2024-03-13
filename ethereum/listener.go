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

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pascaldekloe/etherstream"
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

	sub := e.queryAndConsume(ctx, logger, processingQueue)
	go e.flushMechanism(ctx, logger, processingQueue, sub)
}

func (e *Ethereum) queryAndConsume(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,

) (sub ethereum.Subscription) {

	var err error
	var stream <-chan ethtypes.Log
	var history []ethtypes.Log

	etherReader := etherstream.Reader{Backend: e.wsClient}

	if e.startBlock == 0 {
		e.startBlock = e.latestBlock
	}

	latestBlock := e.latestBlock

	// start initial stream ignoring lookback period
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

	go e.consumeStream(ctx, logger, processingQueue, stream, sub)
	consumeHistroy(logger, history, processingQueue)

	if e.lookbackPeriod == 0 {
		return sub
	}

	// handle lookback period in chunks (some websockets only allow small history queries)
	chunkSize := uint64(100)
	var toUnSub ethereum.Subscription

	for start := (latestBlock - e.lookbackPeriod); start < latestBlock; start += chunkSize {
		end := start + chunkSize
		if end > latestBlock {
			end = latestBlock
		}

		logger.Info(fmt.Sprintf("getting history in chunks: start-block: %d end-block: %d", start, end))

		query = ethereum.FilterQuery{
			Addresses: []common.Address{messageTransmitterAddress},
			Topics:    [][]common.Hash{{messageSent.ID}},
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
		}
		queryAttempt = 1
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
	}

	logger.Info("done querying lookback period. All caught up")

	return sub
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

	var toUnSub ethereum.Subscription
	var history []ethtypes.Log
	var err error

	etherReader := etherstream.Reader{Backend: e.wsClient}
	chunkSize := uint64(100)
	for {
		timer := time.NewTimer(5 * time.Minute)
		select {
		case <-timer.C:
			if e.lastFlushedBlock == 0 {
				e.lastFlushedBlock = e.latestBlock
			}

			latestBlock := e.latestBlock

			for start := (e.lastFlushedBlock - e.lookbackPeriod); start < latestBlock; start += chunkSize {
				end := start + chunkSize
				if end > latestBlock {
					end = latestBlock
				}

				logger.Info(fmt.Sprintf("flushing... querying history in chunks: start-block: %d end-block: %d", start, end))

				query := ethereum.FilterQuery{
					Addresses: []common.Address{messageTransmitterAddress},
					Topics:    [][]common.Hash{{messageSent.ID}},
					FromBlock: big.NewInt(int64(start)),
					ToBlock:   big.NewInt(int64(end)),
				}
				queryAttempt := 1
				for {
					_, toUnSub, history, err = etherReader.QueryWithHistory(ctx, &query)
					if err != nil {
						logger.Error("unable to query history during flush", "err", err)
						queryAttempt++
						time.Sleep(1 * time.Second)
						continue
					}
					break
				}
				toUnSub.Unsubscribe()
				consumeHistroy(logger, history, processingQueue)
			}
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
			}
			e.latestBlock = header.Number.Uint64()

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

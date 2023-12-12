package ethereum

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"math/big"
	"os"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pascaldekloe/etherstream"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

//go:embed abi/MessageTransmitter.json
var content embed.FS

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.TxState) {
	// set up client
	messageTransmitter, err := content.ReadFile("abi/MessageTransmitter.json")
	if err != nil {
		logger.Error("unable to read MessageTransmitter abi", "err", err)
		os.Exit(1)
	}
	messageTransmitterABI, err := abi.JSON(bytes.NewReader(messageTransmitter))
	if err != nil {
		logger.Error("unable to parse MessageTransmitter abi", "err", err)
	}

	messageSent := messageTransmitterABI.Events["MessageSent"]

	ethClient, err := ethclient.DialContext(context.Background(), cfg.Networks.Source.Ethereum.RPC)
	if err != nil {
		logger.Error("unable to initialize ethereum client", "err", err)
		os.Exit(1)
	}

	messageTransmitterAddress := common.HexToAddress(cfg.Networks.Source.Ethereum.MessageTransmitter)
	etherReader := etherstream.Reader{Backend: ethClient}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{messageSent.ID}},
		FromBlock: big.NewInt(int64(cfg.Networks.Source.Ethereum.StartBlock - cfg.Networks.Source.Ethereum.LookbackPeriod)),
	}

	logger.Info(fmt.Sprintf(
		"Starting Ethereum listener at block %d looking back %d blocks",
		cfg.Networks.Source.Ethereum.StartBlock,
		cfg.Networks.Source.Ethereum.LookbackPeriod))

	// websockets do not query history
	// https://github.com/ethereum/go-ethereum/issues/15063
	stream, sub, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	// process history
	for _, historicalLog := range history {
		parsedMsg, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &historicalLog)
		if err != nil {
			logger.Error("Unable to parse history log into MessageState, skipping", "err", err)
			continue
		}
		logger.Info(fmt.Sprintf("New historical msg from source domain %d with tx hash %s", parsedMsg.SourceDomain, parsedMsg.SourceTxHash))

		processingQueue <- &types.TxState{TxHash: parsedMsg.SourceTxHash, Msgs: []*types.MessageState{parsedMsg}}

		// It might help to wait a small amount of time between sending messages into the processing queue
		// so that account sequences / nonces are set correctly
		// time.Sleep(10 * time.Millisecond)
	}

	// consume stream
	go func() {
		var txState *types.TxState
		for {
			select {
			case err := <-sub.Err():
				logger.Error("connection closed", "err", err)
				os.Exit(1)
			case streamLog := <-stream:
				parsedMsg, err := types.EvmLogToMessageState(messageTransmitterABI, messageSent, &streamLog)
				if err != nil {
					logger.Error("Unable to parse ws log into MessageState, skipping")
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
	}()
}

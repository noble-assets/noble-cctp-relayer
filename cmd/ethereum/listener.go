package ethereum

import (
	"bytes"
	"context"
	"cosmossdk.io/log"
	"embed"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pascaldekloe/etherstream"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"math/big"
	"os"
)

//go:embed abi/MessageTransmitter.json
var content embed.FS

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
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
		ToBlock:   big.NewInt(int64(cfg.Networks.Source.Ethereum.StartBlock)),
	}

	// websockets do not query history
	// https://github.com/ethereum/go-ethereum/issues/15063
	stream, sub, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	// process history
	for _, historicalLog := range history {
		parsedMsg, err := types.ToMessageState(messageTransmitterABI, messageSent, &historicalLog)
		if err != nil {
			logger.Error("Unable to parse history log into MessageState, skipping")
			continue
		}
		processingQueue <- parsedMsg
	}

	// consume stream
	go func() {
		for {
			select {
			case err := <-sub.Err():
				logger.Error("connection closed", "err", err)
				os.Exit(1)
			case streamLog := <-stream:
				parsedMsg, err := types.ToMessageState(messageTransmitterABI, messageSent, &streamLog)
				if err != nil {
					logger.Error("Unable to parse ws log into MessageState, skipping")
					continue
				}
				processingQueue <- parsedMsg
			}
		}
	}()
}

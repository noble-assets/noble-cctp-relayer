package ethereum

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/log"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pascaldekloe/etherstream"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

//go:embed abi/MessageTransmitter.json
var content embed.FS

var _ types.Chain = (*Ethereum)(nil)

type Ethereum struct {
	name                      string
	chainID                   int64
	domain                    types.Domain
	rpcURL                    string
	wsURL                     string
	messageTransmitterAddress string
	startBlock                uint64
	lookbackPeriod            uint64
	privateKey                *ecdsa.PrivateKey
	minterAddress             string
	maxRetries                int
	retryIntervalSeconds      int

	mu sync.Mutex
}

func NewChain(
	name string,
	chainID int64,
	domain types.Domain,
	rpcURL string,
	wsURL string,
	messageTransmitterAddress string,
	startBlock uint64,
	lookbackPeriod uint64,
	privateKey string,
	maxRetries int,
	retryIntervalSeconds int,
) (*Ethereum, error) {
	privEcdsaKey, ethereumAddress, err := GetEcdsaKeyAddress(privateKey)
	if err != nil {
		return nil, err
	}
	return &Ethereum{
		name:                      name,
		chainID:                   chainID,
		domain:                    domain,
		rpcURL:                    rpcURL,
		wsURL:                     wsURL,
		messageTransmitterAddress: messageTransmitterAddress,
		startBlock:                startBlock,
		lookbackPeriod:            lookbackPeriod,
		privateKey:                privEcdsaKey,
		minterAddress:             ethereumAddress,
		maxRetries:                maxRetries,
		retryIntervalSeconds:      retryIntervalSeconds,
	}, nil
}

func (e *Ethereum) Name() string {
	return e.name
}

func (e *Ethereum) Domain() types.Domain {
	return e.domain
}

func (e *Ethereum) IsDestinationCaller(destinationCaller []byte) bool {
	zeroByteArr := make([]byte, 32)

	decodedMinter, err := hex.DecodeString(strings.ReplaceAll(e.minterAddress, "0x", ""))
	if err != nil && bytes.Equal(destinationCaller, zeroByteArr) {
		return true
	}

	decodedMinterPadded := make([]byte, 32)
	copy(decodedMinterPadded[12:], decodedMinter)

	return bytes.Equal(destinationCaller, zeroByteArr) || bytes.Equal(destinationCaller, decodedMinterPadded)
}

func (e *Ethereum) InitializeBroadcaster(
	ctx context.Context,
	logger log.Logger,
	sequenceMap *types.SequenceMap,
) error {
	nextNonce, err := GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
	if err != nil {
		return fmt.Errorf("unable to retrieve evm account nonce: %w", err)
	}
	sequenceMap.Put(e.Domain(), uint64(nextNonce))

	return nil
}

func (e *Ethereum) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
	quit chan os.Signal,
) {
	logger = logger.With("chain", e.name, "chain_id", e.chainID, "domain", e.domain)

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

	ethClient, err := ethclient.DialContext(context.Background(), e.wsURL)
	if err != nil {
		logger.Error("unable to initialize ethereum client", "err", err)
		os.Exit(1)
	}

	defer ethClient.Close()

	messageTransmitterAddress := common.HexToAddress(e.messageTransmitterAddress)
	etherReader := etherstream.Reader{Backend: ethClient}

	if e.startBlock == 0 {
		header, err := ethClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			logger.Error("unable to retrieve latest eth block header", "err", err)
			os.Exit(1)
		}

		e.startBlock = header.Number.Uint64()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{messageSent.ID}},
		FromBlock: big.NewInt(int64(e.startBlock - e.lookbackPeriod)),
	}

	logger.Info(fmt.Sprintf(
		"Starting Ethereum listener at block %d looking back %d blocks",
		e.startBlock,
		e.lookbackPeriod))

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
			case <-quit:
				return
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

func (e *Ethereum) Broadcast(
	ctx context.Context,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
) error {

	// set up eth client
	client, err := ethclient.Dial(e.rpcURL)
	if err != nil {
		return fmt.Errorf("unable to dial ethereum client: %w", err)
	}
	defer client.Close()

	backend := NewContractBackendWrapper(client)

	auth, err := bind.NewKeyedTransactorWithChainID(e.privateKey, big.NewInt(e.chainID))
	if err != nil {
		return fmt.Errorf("unable to create auth: %w", err)
	}

	messageTransmitter, err := contracts.NewMessageTransmitter(common.HexToAddress(e.messageTransmitterAddress), backend)
	if err != nil {
		return fmt.Errorf("unable to create message transmitter: %w", err)
	}

	var broadcastErrors error
MsgLoop:
	for _, msg := range msgs {

		if msg.Status == types.Complete {
			continue
		}

		attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
		if err != nil {
			return errors.New("unable to decode message attestation")
		}

		for attempt := 0; attempt <= e.maxRetries; attempt++ {
			logger.Info(fmt.Sprintf(
				"Broadcasting message from %d to %d: with source tx hash %s",
				msg.SourceDomain,
				msg.DestDomain,
				msg.SourceTxHash))

			nonce := sequenceMap.Next(e.domain)
			auth.Nonce = big.NewInt(int64(nonce))

			e.mu.Lock()

			// TODO remove
			nextNonce, err := GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
			if err != nil {
				logger.Error("unable to retrieve account number")
			} else {
				auth.Nonce = big.NewInt(nextNonce)
			}
			// TODO end remove

			// check if nonce already used
			co := &bind.CallOpts{
				Pending: true,
				Context: ctx,
			}

			logger.Debug("Checking if nonce was used for broadcast to Ethereum", "source_domain", msg.SourceDomain, "nonce", msg.Nonce)

			key := append(
				common.LeftPadBytes((big.NewInt(int64(msg.SourceDomain))).Bytes(), 4),
				common.LeftPadBytes((big.NewInt(int64(msg.Nonce))).Bytes(), 8)...,
			)

			response, nonceErr := messageTransmitter.UsedNonces(co, [32]byte(crypto.Keccak256(key)))
			if nonceErr != nil {
				logger.Debug("Error querying whether nonce was used.   Continuing...")
			} else {
				fmt.Printf("received used nonce response: %d\n", response)
				if response.Uint64() == uint64(1) {
					// nonce has already been used, mark as complete
					logger.Debug(fmt.Sprintf("This source domain/nonce has already been used: %d %d",
						msg.SourceDomain, msg.Nonce))
					msg.Status = types.Complete
					e.mu.Unlock()
					continue MsgLoop
				}
			}

			// broadcast txn
			tx, err := messageTransmitter.ReceiveMessage(
				auth,
				msg.MsgSentBytes,
				attestationBytes,
			)
			if err == nil {
				msg.Status = types.Complete

				fullLog, err := tx.MarshalJSON()
				if err != nil {
					logger.Error("error marshalling eth tx log", err)
				}

				msg.DestTxHash = tx.Hash().Hex()

				logger.Info(fmt.Sprintf("Successfully broadcast %s to Ethereum.  Tx hash: %s, FULL LOG: %s", msg.SourceTxHash, msg.DestTxHash, string(fullLog)))
				e.mu.Unlock()
				continue MsgLoop
			}

			logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
			if parsedErr, ok := err.(JsonError); ok {
				if parsedErr.ErrorCode() == 3 && parsedErr.Error() == "execution reverted: Nonce already used" {
					msg.Status = types.Complete
					logger.Error(fmt.Sprintf("This account nonce has already been used: %d", nonce))
					e.mu.Unlock()
					continue MsgLoop
				}

				match, _ := regexp.MatchString("nonce too low: next nonce [0-9]+, tx nonce [0-9]+", parsedErr.Error())
				if match {
					numberRegex := regexp.MustCompile("[0-9]+")
					nextNonce, err := strconv.ParseInt(numberRegex.FindAllString(parsedErr.Error(), 1)[0], 10, 0)
					if err != nil {
						nextNonce, err = GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
						if err != nil {
							logger.Error("unable to retrieve account number")
						}
					}
					sequenceMap.Put(e.domain, uint64(nextNonce))
				}
			}
			e.mu.Unlock()

			// if it's not the last attempt, retry
			// TODO increase the destination.ethereum.broadcast retries (3-5) and retry interval (15s).  By checking for used nonces, there is no gas cost for failed mints.
			if attempt != e.maxRetries {
				logger.Info(fmt.Sprintf("Retrying in %d seconds", e.retryIntervalSeconds))
				time.Sleep(time.Duration(e.retryIntervalSeconds) * time.Second)
			}
		}
		// retried max times with failure
		msg.Status = types.Failed
		broadcastErrors = errors.Join(broadcastErrors, errors.New("reached max number of broadcast attempts"))
	}
	return broadcastErrors
}

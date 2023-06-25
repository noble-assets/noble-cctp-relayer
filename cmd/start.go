package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var (
	MessageTransmitterABI abi.ABI
	TokenMessengerABI     abi.ABI

	DepositForBurn         abi.Event
	DepositForBurnMetadata abi.Event
	DepositForBurnTopics   abi.Arguments
	MessageSent            abi.Event
)

type Message struct {
	message     []byte
	messageHash []byte
	attestation []byte
	isProcessed bool
}

type AttestationResponse struct {
	Attestation string
	Status      string
}

func init() {
	messageTransmitter, _ := os.Open("config/abi/MessageTransmitter.json")
	MessageTransmitterABI, _ = abi.JSON(messageTransmitter)
	tokenMessenger, _ := os.Open("config/abi/TokenMessenger.json")
	TokenMessengerABI, _ = abi.JSON(tokenMessenger)

	DepositForBurn = TokenMessengerABI.Events["DepositForBurn"]
	for _, input := range DepositForBurn.Inputs {
		if input.Indexed {
			DepositForBurnTopics = append(DepositForBurnTopics, input)
		}
	}
	MessageSent = MessageTransmitterABI.Events["MessageSent"]

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions between Ethereum and Noble",
	Run:   start,
}

// txMap maps eth tx_hash to message metadata
var txMap = map[string]Message{}

// currentBlock marks the next sequential ethereum block to be processed
var currentBlock *big.Int

func start(cmd *cobra.Command, args []string) {
	currentBlock = big.NewInt(cfg.Indexer.StartBlock)
	client, err := ethclient.Dial(cfg.Networks.Ethereum.RPC)
	if err != nil {
		logger.Error("unable to initialise client", "err", err)
		os.Exit(1)
	}

	heightChan := make(chan int64, 10000)
	threads := 128
	wg := sync.WaitGroup{}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processBlock(client, heightChan)
		}()
	}

	enqueueBlockHeights(client, heightChan)

	wg.Wait()
}

func enqueueBlockHeights(client *ethclient.Client, heightChan chan int64) {
	//for {
	//	// add blocks to queue
	//	if len(heightChan) < cap(heightChan)/4 {
	//		latestBlock, _ := client.BlockNumber(context.Background())
	//
	//		for currentBlock.Uint64() <= latestBlock && len(heightChan) != cap(heightChan) {
	//			heightChan <- currentBlock.Int64()
	//			currentBlock = currentBlock.Add(currentBlock, big.NewInt(1))
	//		}
	//
	//	}
	//	time.Sleep(20 * time.Second)
	//}
	heightChan <- 17549952
}

// TODO(@john): Handle all errors.
func processBlock(client *ethclient.Client, heightChan chan int64) {
	for {
		rawHeight := <-heightChan
		height := big.NewInt(rawHeight)
		logger.Debug("processing new block", "height", height)

		// Fetch entire block by height, giving us the block hash.
		block, _ := client.BlockByNumber(context.Background(), height)
		hash := block.Hash()

		// Query relevant logs for this specific block.
		filter := ethereum.FilterQuery{
			BlockHash: &hash,
			Addresses: []common.Address{TokenMessenger},
			Topics:    [][]common.Hash{{DepositForBurn.ID}},
		}
		logs, _ := client.FilterLogs(context.Background(), filter)

		// ...
		for _, log := range logs {
			// Ensure the burn token is correct.
			// NOTE: The first topic is the event name, so we ignore it.
			topics := make(map[string]interface{})
			_ = abi.ParseTopicsIntoMap(topics, DepositForBurnTopics, log.Topics[1:])

			isValidBurnToken := ValidTokens[topics["burnToken"].(common.Address)]
			if !isValidBurnToken {
				continue
			}

			// Ensure the destination domain is correct.
			event := make(map[string]interface{})
			_ = TokenMessengerABI.UnpackIntoMap(event, DepositForBurn.Name, log.Data)

			if event["destinationDomain"] != cfg.Networks.Noble.DestinationId {
				continue
			}

			// ...
			_, found := txMap[log.TxHash.String()]
			if !found {
				receipt, _ := client.TransactionReceipt(context.Background(), log.TxHash)

				messageSentEvent := make(map[string]interface{})

				for _, log := range receipt.Logs {
					// NOTE: The first topic is the event name, so we filter it.
					switch log.Topics[0] {
					case DepositForBurnMetadata.ID:
						// TODO(@john): Handle IBC Metadata.
					case MessageSent.ID:
						if log.Address != MessageTransmitter {
							continue
						}

						_ = MessageTransmitterABI.UnpackIntoMap(messageSentEvent, MessageSent.Name, log.Data)
					}
				}

				messageHash := crypto.Keccak256Hash(messageSentEvent["message"].([]uint8)).Bytes()
				txMap[log.TxHash.String()] = Message{
					messageHash: messageHash,
				}
			}
		}

		// mints to broadcast
		// broadcastQueue := make(chan Message)
		// look up attestations for all unprocessed blocks

		for txHash, tx := range txMap {
			if !tx.isProcessed {
				rawResponse, _ := http.Get(cfg.Indexer.AttestationBaseUrl + "0x" + hex.EncodeToString(tx.messageHash))
				body, _ := io.ReadAll(rawResponse.Body)

				response := AttestationResponse{}
				_ = json.Unmarshal(body, &response)

				logger.Info("queried attestation", "tx", txHash, "status", response.Status)

				if response.Status == "complete" {
					tx.attestation = []byte(response.Attestation)
					// broadcastQueue <- pendingTx
				}
			}
		}

		// time.Sleep(1 * time.Second)

		// TODO async broadcast all messages from broadcastQueue to Noble and mark as processed

		// TODO query every blockResponse on Noble to evict successfully relayed messages from cache
	}
}

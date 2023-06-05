package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type MintMessage struct {
	message     []byte
	messageHash []byte
	attestation []byte
	isProcessed bool
}

func (m MintMessage) String() string {
	isProcessed := "false"
	if m.isProcessed {
		isProcessed = "true"
	}
	return fmt.Sprintf("{message: %s, messageHash: %s, attestation: %s, isProcessed: %s",
		"0x"+hex.EncodeToString(m.message), "0x"+hex.EncodeToString(m.messageHash), "0x"+hex.EncodeToString(m.attestation), isProcessed)
}

type AttestationResponse struct {
	Attestation string
	Status      string
}

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Long:  `Start relaying CCTP transactions from Ethereum to Noble.`,
	Run:   start,
}

// txMap maps eth tx_hash to message metadata
var txMap = map[string]MintMessage{}

// currentBlock marks the next sequential ethereum block to be processed
var currentBlock *big.Int

func start(cmd *cobra.Command, args []string) {

	currentBlock = big.NewInt(conf.Indexer.StartBlock)
	client, err := ethclient.Dial(conf.Networks.Ethereum.RPC)
	if err != nil {
		log.Fatal(err)
	}

	// load abi
	tokenMessengerJson, err := os.ReadFile("config/abi/TokenMessenger.json")
	tokenMessengerAbi, err := abi.JSON(strings.NewReader(string(tokenMessengerJson)))
	messageTransmitterJson, err := os.ReadFile("config/abi/MessageTransmitter.json")
	messageTransmitterAbi, err := abi.JSON(strings.NewReader(string(messageTransmitterJson)))
	if err != nil {
		log.Fatal(err)
	}

	heightChan := make(chan int64, 10000)
	threads := 128
	wg := sync.WaitGroup{}

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processBlock(client, heightChan, tokenMessengerAbi, messageTransmitterAbi)
		}()
	}

	enqueueBlockHeights(client, heightChan)

	wg.Wait()
}

func enqueueBlockHeights(client *ethclient.Client, heightChan chan int64) {
	for {
		// add blocks to queue
		if len(heightChan) < cap(heightChan)/4 {
			latest, err := client.HeaderByNumber(context.Background(), nil)
			if err != nil {
				log.Fatal(err)
			}
			latestBlock := latest.Number

			for currentBlock.Int64() <= latestBlock.Int64() && len(heightChan) != cap(heightChan) {
				heightChan <- currentBlock.Int64()
				currentBlock = currentBlock.Add(currentBlock, big.NewInt(1))
			}

		}
		time.Sleep(20 * time.Second)
	}
}

func processBlock(client *ethclient.Client, heightChan chan int64, tokenMessengerAbi abi.ABI, messageTransmitterAbi abi.ABI) {
	for {
		rawHeight := <-heightChan
		block := big.NewInt(rawHeight)
		log.Printf("Processing blockResponse %s", block.String())

		blockResponse, err := client.BlockByNumber(context.Background(), block)
		fmt.Println("Block response received from blockResponse " + blockResponse.Number().String())
		if err != nil && err.Error() == "not found" {
			log.Println("Block not found")
			time.Sleep(5 * time.Second)
		} else if err != nil {
			log.Fatal(err)
		}

		for _, tx := range blockResponse.Transactions() {
			if isDepositForBurnTx(tx, &tokenMessengerAbi) {
				_, found := txMap[tx.Hash().String()]
				if !found {
					// get keccack-256 hash of MessageSent event
					receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
					if err != nil {
						log.Fatal(err)
					}

					for _, vLog := range receipt.Logs {

						// topic[0] is the event name
						event, err := messageTransmitterAbi.EventByID(vLog.Topics[0])

						if err != nil || event.Name != "MessageSent" {
							continue
						}

						if len(vLog.Data) > 0 {
							outputDataMap := make(map[string]interface{})
							err = messageTransmitterAbi.UnpackIntoMap(outputDataMap, event.Name, vLog.Data)
							if err != nil {
								log.Printf("Unable to parse tx messages for tx hash: %s", tx.Hash().String())
								continue
							}

							messageHash := crypto.Keccak256Hash(outputDataMap["message"].([]uint8)).Bytes()
							txMap[tx.Hash().String()] = MintMessage{
								message:     tx.Data(),
								messageHash: messageHash,
								attestation: nil,
								isProcessed: false,
							}
						}

					}

				}
			}
		}

		// mints to broadcast
		// broadcastQueue := make(chan MintMessage)
		// look up attestations for all unprocessed blocks

		for _, pendingTx := range txMap {
			if !pendingTx.isProcessed {
				resp, err := http.Get(conf.Indexer.AttestationBaseUrl + "0x" + hex.EncodeToString(pendingTx.messageHash))
				if err != nil {
					fmt.Println("Failed to look up attestation with message hash 0x" + hex.EncodeToString(pendingTx.messageHash))
					continue
				}

				body, _ := io.ReadAll(resp.Body)
				response := AttestationResponse{}
				err = json.Unmarshal(body, &response)
				if err != nil {
					fmt.Println("Failure to parse response body.")
				}

				if resp.StatusCode == 200 && response.Status == "complete" {
					pendingTx.attestation = []byte(response.Attestation)
					// broadcastQueue <- pendingTx

					pendingTx.isProcessed = true

				}
			}
		}

		//time.Sleep(1 * time.Second)

		// TODO async broadcast all messages from broadcastQueue to Noble and mark as processed

		// TODO query every blockResponse on Noble to evict successfully relayed messages from cache
	}
}

// returns true if tx is a depositForBurn or depositForBurnWithCaller txn from TokenMessenger
func isDepositForBurnTx(tx *types.Transaction, contractAbi *abi.ABI) bool {
	if tx == nil || tx.Data() == nil || tx.To() == nil || tx.To().String() != conf.Networks.Ethereum.TokenMessenger {
		return false
	}

	method, inputs := DecodeTransactionInputData(contractAbi, tx.Data())
	if method != "depositForBurn" && method != "depositForBurnWithCaller" {
		return false
	}

	_, destinationDomainFound := inputs["destinationDomain"]
	if !destinationDomainFound || inputs["destinationDomain"].(uint32) != conf.Networks.Noble.DestinationId {
		return false
	}

	// check that it is relaying correct tokens
	burnToken, burnTokenFound := inputs["burnToken"]
	isValidTokenAddress := conf.Indexer.ValidTokenAddresses[burnToken.(common.Address).String()]

	if !burnTokenFound || !isValidTokenAddress {
		return false
	}

	return true
}

func DecodeTransactionInputData(contractABI *abi.ABI, data []byte) (string, map[string]interface{}) {
	methodSigData := data[:4]
	inputsSigData := data[4:]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}
	inputsMap := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		log.Fatal(err)
	}

	return method.Name, inputsMap
}

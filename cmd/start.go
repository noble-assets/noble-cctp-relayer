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

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Long:  `Start relaying CCTP transactions from Ethereum to Noble.`,
	Run:   start,
}

// TX_MAP maps eth tx_hash to message bytes, attestation
var TX_MAP = map[string]MintMessage{}

func start(cmd *cobra.Command, args []string) {

	currentBlock := big.NewInt(conf.Indexer.StartBlock)

	// start webserver
	go func() {

		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			responseMap := map[string]string{}
			responseMap["Current block"] = currentBlock.String()
			for key, val := range TX_MAP {
				responseMap[key] = val.String()
			}

			response, _ := json.Marshal(responseMap)

			fmt.Fprintf(w, string(response))
		})

		if err := http.ListenAndServe(":80", nil); err != nil {
			log.Fatal(err)
		}

	}()

	// Connect to an Ethereum client
	client, err := ethclient.Dial(conf.Networks.Ethereum.RPC)
	if err != nil {
		log.Fatal(err)
	}

	tokenMessengerJson, err := os.ReadFile("config/abi/TokenMessenger.json")
	if err != nil {
		fmt.Println(err)
	}

	tokenMessengerAbi, err := abi.JSON(strings.NewReader(string(tokenMessengerJson)))
	if err != nil {
		log.Fatal(err)
	}

	messageTransmitterJson, err := os.ReadFile("config/abi/MessageTransmitter.json")
	if err != nil {
		fmt.Println(err)
	}

	messageTransmitterAbi, err := abi.JSON(strings.NewReader(string(messageTransmitterJson)))

	for {
		block, err := client.BlockByNumber(context.Background(), currentBlock)
		if err != nil {
			log.Fatal(err)
		}

		for _, tx := range block.Transactions() {
			if isDepositForBurnTx(tx, &tokenMessengerAbi) {
				_, found := TX_MAP[tx.Hash().String()]
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
								log.Fatal(err)
							}

							messageHash := crypto.Keccak256Hash(outputDataMap["message"].([]uint8)).Bytes()
							TX_MAP[tx.Hash().String()] = MintMessage{
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

		// look up attestations for all unprocessed blocks
		broadcastQueue := make(chan MintMessage)
		for _, pendingTx := range TX_MAP {
			if !pendingTx.isProcessed {
				resp, err := http.Get(conf.Indexer.AttestationBaseUrl + "0x" + hex.EncodeToString(pendingTx.messageHash))
				if err != nil {
					fmt.Println("Failed to look up attestation with message hash 0x" + hex.EncodeToString(pendingTx.messageHash))
				}
				defer resp.Body.Close()

				type AttestationResponse struct {
					Attestation string
					Status      string
				}

				body, _ := io.ReadAll(resp.Body)
				response := AttestationResponse{}
				err = json.Unmarshal(body, &response)
				if err != nil {
					fmt.Println("Failure to parse response body.")
				}

				if resp.StatusCode == 200 && response.Status == "complete" {
					pendingTx.attestation = []byte(response.Attestation)
					broadcastQueue <- pendingTx

					pendingTx.isProcessed = true

				}
			}
		}

		// TODO async broadcast all messages from broadcastQueue to Noble and mark as processed

		// query every block on Noble to evict successfully relayed messages from cache
		// TODO add this in once module is live on Noble testnet

		currentBlock = currentBlock.Add(currentBlock, big.NewInt(1))
	}

}

// returns true if tx is a depositForBurn or depositForBurnWithCaller txn from TokenMessenger
func isDepositForBurnTx(tx *types.Transaction, contractAbi *abi.ABI) bool {
	if tx == nil || tx.Data() == nil || tx.To() == nil {
		return false
	}

	if tx.To().String() != conf.Networks.Ethereum.TokenMessenger {
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

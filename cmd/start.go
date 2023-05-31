package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Long:  `Start relaying CCTP transactions from Ethereum to Noble.`,
	Run:   start,
}

const START_BLOCK = 17312317
const TOKEN_MESSENGER = "0xBd3fa81B58Ba92a82136038B25aDec7066af3155"
const MESSAGE_TRANSMITTER = "0x0a992d191deec32afe36203ad87d7d289a738f81"
const NOBLE_DESTINATION_ID = 1 // TODO what is this for noble?
const ATTESTATION_BASE_URL = "https://iris-api.circle.com/attestations/"

// VALID_TOKEN_ADDRESSES contains valid eth token addresses to mint
var VALID_TOKEN_ADDRESSES = map[string]bool{"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48": true} // just usdc

// TX_MAP maps eth tx_hash to message bytes, attestation
var TX_MAP = map[string]MintMessage{}

func start(cmd *cobra.Command, args []string) {

	// Connect to an Ethereum client
	client, err := ethclient.Dial(conf.Networks.Ethereum.RPC)
	if err != nil {
		log.Fatal(err)
	}

	json, err := os.ReadFile("config/abi/TokenMessenger.json")
	if err != nil {
		fmt.Println(err)
	}

	tokenMessengerAbi, err := abi.JSON(strings.NewReader(string(json)))
	if err != nil {
		log.Fatal(err)
	}

	json, err = os.ReadFile("config/abi/MessageTransmitter.json")
	if err != nil {
		fmt.Println(err)
	}

	messageTransmitterAbi, err := abi.JSON(strings.NewReader(string(json)))

	currentBlock := big.NewInt(START_BLOCK)
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
						fmt.Println(vLog.Address.Hex())
						fmt.Println(vLog.TxHash.Hex())

						// topic[0] is the event name
						event, err := messageTransmitterAbi.EventByID(vLog.Topics[0])

						if err != nil || event.Name != "MessageSent" {
							continue
						}

						if len(vLog.Data) > 0 {
							fmt.Printf("Log Data in Hex: %s\n", hex.EncodeToString(vLog.Data))
							outputDataMap := make(map[string]interface{})
							err = messageTransmitterAbi.UnpackIntoMap(outputDataMap, event.Name, vLog.Data)
							if err != nil {
								log.Fatal(err)
							}
							fmt.Printf("Event outputs: %v\n", outputDataMap)

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
		for _, pendingTx := range TX_MAP {
			if pendingTx.isProcessed {
				// evict
			} else {
				resp, err := http.Get(ATTESTATION_BASE_URL + string(pendingTx.messageHash))
				if err != nil {
					// TODO failed to look up attestation
				}
				defer resp.Body.Close()
				//body, err := io.ReadAll(resp.Body)

			}
		}

		// TODO query every block on Noble to evict successfully relayed messages from cache

		currentBlock = currentBlock.Add(currentBlock, big.NewInt(1))
	}

}

// returns true if tx is a depositForBurn or depositForBurnWithCaller txn from TokenMessenger
func isDepositForBurnTx(tx *types.Transaction, contractAbi *abi.ABI) bool {
	if tx.To().String() != TOKEN_MESSENGER {
		return false
	}

	// decode tx params
	method, inputs := DecodeTransactionInputData(contractAbi, tx.Data())
	if method != "depositForBurn" && method != "depositForBurnWithCaller" {
		return false
	}

	_, destinationDomainFound := inputs["destinationDomain"]
	if !destinationDomainFound || inputs["destinationDomain"].(uint32) != NOBLE_DESTINATION_ID {
		return false
	}

	// check that it is relaying correct tokens
	burnToken, burnTokenFound := inputs["burnToken"]
	isValidTokenAddress := VALID_TOKEN_ADDRESSES[burnToken.(common.Address).String()]

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
	} else {
		fmt.Println(inputsMap)
	}

	return method.Name, inputsMap
}

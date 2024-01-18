package cmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

type Processor struct {
	Mu sync.RWMutex
}

func NewProcessor() *Processor {
	return &Processor{}
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Run:   Start,
}

// State and Store map the iris api lookup id -> MessageState
// State represents all in progress burns/mints
// Store represents terminal states
var State = types.NewStateMap()

// SequenceMap maps the domain -> the equivalent minter account sequence or nonce
var sequenceMap = types.NewSequenceMap()

func Start(cmd *cobra.Command, args []string) {

	p := NewProcessor()

	var wg sync.WaitGroup
	wg.Add(1)

	// initialize minter account sequences
	for key := range Cfg.Networks.Minters {
		switch key {
		case 0:
			ethNonce, err := ethereum.GetEthereumAccountNonce(
				Cfg.Networks.Destination.Ethereum.RPC,
				Cfg.Networks.Minters[0].MinterAddress)

			if err != nil {
				Logger.Error("Error retrieving Ethereum account nonce")
				os.Exit(1)
			}
			sequenceMap.Put(key, ethNonce)
		case 4:
			_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
				Cfg.Networks.Destination.Noble.API,
				Cfg.Networks.Minters[4].MinterAddress)

			if err != nil {
				Logger.Error("Error retrieving Noble account sequence")
				os.Exit(1)
			}
			sequenceMap.Put(key, nextMinterSequence)
		}

		// ...initialize more here
	}

	// messageState processing queue
	var processingQueue = make(chan *types.TxState, 10000)

	// spin up Processor worker pool
	for i := 0; i < int(Cfg.ProcessorWorkerCount); i++ {
		go p.StartProcessor(cmd.Context(), Cfg, Logger, processingQueue, sequenceMap)
	}

	// listeners listen for events, parse them, and enqueue them to processingQueue
	if Cfg.Networks.Source.Ethereum.Enabled {
		ethereum.StartListener(Cfg, Logger, processingQueue)
	}
	if Cfg.Networks.Source.Noble.Enabled {
		noble.StartListener(Cfg, Logger, processingQueue)
	}
	// ...register more chain listeners here

	wg.Wait()
}

// StartProcessor is the main processing pipeline.
func (p *Processor) StartProcessor(ctx context.Context, cfg config.Config, logger log.Logger, processingQueue chan *types.TxState, sequenceMap *types.SequenceMap) {
	for {
		dequeuedTx := <-processingQueue

		// if this is the first time seeing this message, add it to the State
		tx, ok := State.Load(LookupKey(dequeuedTx.TxHash))
		if !ok {
			State.Store(LookupKey(dequeuedTx.TxHash), dequeuedTx)
			tx, _ = State.Load(LookupKey(dequeuedTx.TxHash))
			for _, msg := range tx.Msgs {
				msg.Status = types.Created
			}
		}

		var broadcastMsgs = make(map[types.Domain][]*types.MessageState)
		var requeue bool
		for _, msg := range tx.Msgs {

			// if a filter's condition is met, mark as filtered
			if filterDisabledCCTPRoutes(cfg, logger, msg) ||
				filterInvalidDestinationCallers(cfg, logger, msg) {
				msg.Status = types.Filtered
			}

			// if the message is burned or pending, check for an attestation
			if msg.Status == types.Created || msg.Status == types.Pending {
				response := circle.CheckAttestation(cfg, logger, msg.IrisLookupId, msg.SourceTxHash, msg.SourceDomain, msg.DestDomain)
				if response != nil {
					if msg.Status == types.Created && response.Status == "pending_confirmations" {
						logger.Debug("Attestation is created but still pending confirmations for 0x" + msg.IrisLookupId + ".  Retrying...")
						msg.Status = types.Pending
						msg.Updated = time.Now()
						time.Sleep(10 * time.Second)
						requeue = true
						continue
					} else if response.Status == "pending_confirmations" {
						logger.Debug("Attestation is still pending for 0x" + msg.IrisLookupId + ".  Retrying...")
						time.Sleep(10 * time.Second)
						requeue = true
						continue
					} else if response.Status == "complete" {
						logger.Debug("Attestation is complete for 0x" + msg.IrisLookupId + ".  Retrying...")
						msg.Status = types.Attested
						msg.Attestation = response.Attestation
						msg.Updated = time.Now()
						broadcastMsgs[msg.DestDomain] = append(broadcastMsgs[msg.DestDomain], msg)
					}
				} else {
					// add attestation retry intervals per domain here
					logger.Debug("Attestation is still processing for 0x" + msg.IrisLookupId + ".  Retrying...")
					time.Sleep(10 * time.Second)
					// retry
					requeue = true
					continue
				}
			}
		}
		// if the message is attested to, try to broadcast
		for domain, msgs := range broadcastMsgs {
			var err error
			switch domain {
			case 0: // ethereum
				err = ethereum.Broadcast(ctx, cfg, logger, msgs, sequenceMap)
			case 4: // noble
				err = noble.Broadcast(ctx, cfg, logger, msgs, sequenceMap)
			}

			if err != nil {
				// TODO: add dest domain to error log
				logger.Error("unable to mint one or more transfers", "error(s)", err, "total_transfers", len(msgs))
				requeue = true
				continue
			}
			// ...add minters for different domains here

			for _, msg := range msgs {
				msg.Status = types.Complete
				msg.Updated = time.Now()
			}

		}
		if requeue {
			processingQueue <- tx
		}
	}
}

// filterDisabledCCTPRoutes returns true if we haven't enabled relaying from a source domain to a destination domain
func filterDisabledCCTPRoutes(cfg config.Config, logger log.Logger, msg *types.MessageState) bool {
	val, ok := cfg.Networks.EnabledRoutes[msg.SourceDomain]
	result := !(ok && val == msg.DestDomain)
	if result {
		logger.Info(fmt.Sprintf("Filtered tx %s because relaying from %d to %d is not enabled",
			msg.SourceTxHash, msg.SourceDomain, msg.DestDomain))
	}
	return result
}

// filterInvalidDestinationCallers returns true if the minter is not the destination caller for the specified domain
func filterInvalidDestinationCallers(cfg config.Config, logger log.Logger, msg *types.MessageState) bool {
	zeroByteArr := make([]byte, 32)
	result := false

	switch msg.DestDomain {
	case 4:
		bech32DestinationCaller, err := types.DecodeDestinationCaller(msg.DestinationCaller)
		if err != nil {
			result = true
		}
		if !bytes.Equal(msg.DestinationCaller, zeroByteArr) &&
			bech32DestinationCaller != cfg.Networks.Minters[msg.DestDomain].MinterAddress {
			result = true
		}
		if result {
			logger.Info(fmt.Sprintf("Filtered tx %s because the destination caller %s is specified and it's not the minter %s",
				msg.SourceTxHash, msg.DestinationCaller, cfg.Networks.Minters[msg.DestDomain].MinterAddress))
		}

	default: // minting to evm
		decodedMinter, err := hex.DecodeString(strings.ReplaceAll(cfg.Networks.Minters[0].MinterAddress, "0x", ""))
		if err != nil {
			return !bytes.Equal(msg.DestinationCaller, zeroByteArr)
		}

		decodedMinterPadded := make([]byte, 32)
		copy(decodedMinterPadded[12:], decodedMinter)

		if !bytes.Equal(msg.DestinationCaller, zeroByteArr) && !bytes.Equal(msg.DestinationCaller, decodedMinterPadded) {
			result = true
		}
	}

	return result
}

func LookupKey(sourceTxHash string) string {
	// return fmt.Sprintf("%s-%s", sourceTxHash, messageType)
	return sourceTxHash
}

func init() {
	cobra.OnInitialize(func() {})
}

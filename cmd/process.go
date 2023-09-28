package cmd

import (
	"bytes"
	"cosmossdk.io/log"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"os"
	"sync"
	"time"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Run:   Start,
}

// State and Store map the iris api lookup id -> MessageState
// State represents all in progress burns/mints
// Store represents terminal states
var State = types.NewStateMap()

// SequenceMap maps the domain -> the equivalent minter address sequence/nonce
var sequenceMap = types.NewSequenceMap()

func Start(cmd *cobra.Command, args []string) {

	var wg sync.WaitGroup
	wg.Add(1)

	// initialize minter account sequences
	for key, _ := range Cfg.Networks.Minters {
		switch key {
		case 4:
			_, nextMinterSequence, err := noble.GetNobleAccountNumberSequence(
				Cfg.Networks.Destination.Noble.API,
				Cfg.Networks.Minters[4].MinterAddress)

			if err != nil {
				Logger.Error("Error retrieving account sequence")
				os.Exit(1)
			}
			sequenceMap.Put(key, nextMinterSequence)
		}

		// ...initialize more here
	}

	// messageState processing queue
	var processingQueue = make(chan *types.MessageState, 10000)

	// spin up Processor worker pool
	for i := 0; i < int(Cfg.ProcessorWorkerCount); i++ {
		go StartProcessor(Cfg, Logger, processingQueue, sequenceMap)
	}

	// listeners listen for events, parse them, and enqueue them to processingQueue
	if Cfg.Networks.Source.Ethereum.Enabled {
		ethereum.StartListener(Cfg, Logger, processingQueue)
	}
	// ...register more chain listeners here

	wg.Wait()
}

// StartProcessor is the main processing pipeline.
func StartProcessor(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState, sequenceMap *types.SequenceMap) {
	for {
		dequeuedMsg := <-processingQueue
		// if this is the first time seeing this message, add it to the State
		msg, ok := State.Load(LookupKey(dequeuedMsg.SourceTxHash, dequeuedMsg.Type))
		if !ok {
			State.Store(LookupKey(dequeuedMsg.SourceTxHash, dequeuedMsg.Type), dequeuedMsg)
			msg, _ = State.Load(LookupKey(dequeuedMsg.SourceTxHash, dequeuedMsg.Type))
			msg.Status = types.Created
		}

		// if a filter's condition is met, mark as filtered
		if filterDisabledCCTPRoutes(cfg, msg) ||
			filterInvalidDestinationCallers(cfg, msg) ||
			filterNonWhitelistedChannels(cfg, msg) {
			msg.Status = types.Filtered
		}

		// if the message is burned or pending, check for an attestation
		if msg.Status == types.Created || msg.Status == types.Pending {
			response := circle.CheckAttestation(cfg, logger, msg.IrisLookupId)
			if response != nil {
				if msg.Status == types.Created && response.Status == "pending_confirmations" {
					msg.Status = types.Pending
					msg.Updated = time.Now()
					time.Sleep(10 * time.Second)
					processingQueue <- msg
					continue
				} else if response.Status == "pending_confirmations" {
					time.Sleep(10 * time.Second)
					processingQueue <- msg
					continue
				} else if response.Status == "complete" {
					msg.Status = types.Attested
					msg.Attestation = response.Attestation
					msg.Updated = time.Now()
				}
			} else {
				// add attestation retry intervals per domain here
				logger.Debug("Attestation is still processing for 0x" + msg.IrisLookupId + ".  Retrying...")
				time.Sleep(10 * time.Second)
				// retry
				processingQueue <- msg
				continue
			}
		}
		// if the message is attested to, try to broadcast
		if msg.Status == types.Attested {
			switch msg.DestDomain {
			case 4: // noble
				response, err := noble.Broadcast(cfg, logger, msg, sequenceMap)
				if err != nil {
					logger.Error("unable to mint", "err", err)
					processingQueue <- msg
					continue
				}
				if response.Code != 0 {
					logger.Error("nonzero response code received", "err", err)
					processingQueue <- msg
					continue
				}
				// success!
				msg.DestTxHash = response.Hash.String()
			}
			// ...add minters for different domains here

			msg.Status = types.Complete
			msg.Updated = time.Now()
		}
	}
}

// filterDisabledCCTPRoutes returns true if we haven't enabled relaying from a source domain to a destination domain
func filterDisabledCCTPRoutes(cfg config.Config, msg *types.MessageState) bool {
	val, ok := cfg.Networks.EnabledRoutes[msg.SourceDomain]
	return !(ok && val == msg.DestDomain)
}

// filterInvalidDestinationCallers returns true if the minter is not the destination caller for the specified domain
func filterInvalidDestinationCallers(cfg config.Config, msg *types.MessageState) bool {
	zeroByteArr := make([]byte, 32)
	bech32DestinationCaller, err := types.DecodeDestinationCaller(msg.DestinationCaller)
	if err != nil {
		return true
	}
	if !bytes.Equal(msg.DestinationCaller, zeroByteArr) &&
		bech32DestinationCaller != cfg.Networks.Minters[msg.DestDomain].MinterAddress {
		return true
	}
	return false
}

// filterNonWhitelistedChannels is a Noble specific filter that returns true
// if the channel is not in the forwarding_channel_whitelist
func filterNonWhitelistedChannels(cfg config.Config, msg *types.MessageState) bool {
	if !cfg.Networks.Destination.Noble.FilterForwardsByIbcChannel {
		return false
	}
	for _, channel := range cfg.Networks.Destination.Noble.ForwardingChannelWhitelist {
		if msg.Channel == channel {
			return false
		}
	}
	return true
}

func LookupKey(sourceTxHash string, messageType string) string {
	return fmt.Sprintf("%s-%s", sourceTxHash, messageType)
}

func init() {
	cobra.OnInitialize(func() {})
}

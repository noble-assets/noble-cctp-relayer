package cmd

import (
	"bytes"
	"cosmossdk.io/log"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
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
var State = map[string]types.MessageState{}
var Store = map[string]types.MessageState{}

func Start(cmd *cobra.Command, args []string) {

	// messageState processing queue
	var processingQueue = make(chan *types.MessageState, 10000)

	// listeners listen for events, parse them, and enqueue them to processingQueue
	if Cfg.Networks.Source.Ethereum.Enabled {
		go ethereum.StartListener(Cfg, Logger, processingQueue)
	}
	// ...register more chain listeners here

	// spin up Processor worker pool
	for i := 0; i < int(Cfg.ProcessorWorkerCount); i++ {
		go StartProcessor(Cfg, Logger, processingQueue)
	}

	// constantly cycle through active MessageStates
	for {
		for _, msg := range State {
			processingQueue <- &msg
		}
	}
}

// StartProcessor is the main processing pipeline.
func StartProcessor(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
	for {
		msg := <-processingQueue
		// if this is the first time seeing this message, add it to the State
		if _, ok := State[msg.IrisLookupId]; !ok {
			msg.Status = types.Created
			State[msg.IrisLookupId] = *msg
		}

		// filters return true if they meet a condition
		if filterDisabledCCTPRoutes(cfg, msg) ||
			filterInvalidDestinationCallers(cfg, msg) ||
			filterNonWhitelistedChannels(cfg, msg) {
			msg.Status = types.Filtered
		}

		// if the message is burned or pending, check for an attestation
		if msg.Status == types.Created || msg.Status == types.Pending {
			response, exists := circle.CheckAttestation(cfg, logger, msg.IrisLookupId)
			if exists {
				if msg.Status == types.Created && response.Status == "pending" {
					msg.Status = types.Pending
					msg.Updated = time.Now()
					return
				} else if response.Status == "complete" {
					msg.Status = types.Attested
					msg.Updated = time.Now()
				}
			} else {
				// add attestation retry intervals per domain here
				time.Sleep(30 * time.Second)
				return
			}
		}
		// if the message is attested to, try to mint
		if msg.Status == types.Attested {
			switch msg.DestDomain {
			case 4: // noble
				response, err := noble.Broadcast(cfg, logger, *msg)
				if err != nil {
					logger.Error("unable to mint", "err", err)
					return
				}
				if response.Code != 0 {
					logger.Error("nonzero response code received", "err", err)
					return
				}
				// success!
				msg.DestTxHash = response.TxHash
			}
			// ...add minters for different domains here

			msg.Status = types.Complete
			msg.Updated = time.Now()
		}

		// remove messages with terminal State, add to Store
		if msg.Status == types.Complete || msg.Status == types.Failed || msg.Status == types.Filtered {
			delete(State, msg.IrisLookupId)
			Store[msg.IrisLookupId] = *msg
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

func init() {
	cobra.OnInitialize(func() {})
}

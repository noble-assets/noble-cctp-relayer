package cmd

import (
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/ethereum"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"time"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Run:   Start,
}

// iris api lookup id -> MessageState - in progress states
var state = map[string]types.MessageState{}

// iris api lookup id -> MessageState - terminal states
var store = map[string]types.MessageState{}

// messageState processing queue
var processingQueue = make(chan *types.MessageState, 10000)

func Start(cmd *cobra.Command, args []string) {

	// listeners listen for events, parse them, and enqueue them to processingQueue
	if Cfg.Networks.Source.Ethereum.Enabled {
		go ethereum.StartListener(Cfg, Logger, processingQueue)
	}
	// ...register more chain listeners here

	// spin up StartProcessor worker pool
	for i := 0; i < 4; i++ {
		go StartProcessor()
	}

	// constantly cycle through MessageStates
	for {
		for _, msg := range state {
			processingQueue <- &msg
		}
	}
}

// StartProcessor is the main processing pipeline.
func StartProcessor() {
	for {
		msg := <-processingQueue
		// if this is the first time seeing this message, add it to the state
		if _, ok := state[msg.IrisLookupId]; !ok {
			state[msg.IrisLookupId] = *msg
		}

		// filters
		if msg.FilterDisabledCCTPRoutes(Cfg.EnabledRoutes) ||
			msg.FilterInvalidDestinationCallers(Cfg.Minters[msg.DestDomain].MinterAddress) ||
			msg.FilterNonWhitelistedChannels(
				Cfg.Networks.Destination.Noble.FilterForwardsByIbcChannel,
				Cfg.Networks.Destination.Noble.ForwardingChannelWhitelist) {
			msg.Status = types.Filtered
		}

		// if the message is burned or pending, check for an attestation
		if msg.Status == types.Created || msg.Status == types.Pending {
			response, exists := circle.CheckAttestation(Cfg, Logger, msg.IrisLookupId)
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
				time.Sleep(30 * time.Second)
				return
			}
		}
		// if the message is attested to, try to mint
		if msg.Status == types.Attested {
			switch msg.DestDomain {
			case 0:
				response, err := noble.Broadcast(Cfg, Logger, msg)
				if err != nil {
					Logger.Error("unable to mint", "err", err)
					return
				}
				if response.Code != 0 {
					Logger.Error("nonzero response code received", "err", err)
					return
				}
				// success!
				msg.DestTxHash = response.TxHash
			}
			msg.Status = types.Complete
			msg.Updated = time.Now()
		}

		// remove terminal states from state, add to store
		if msg.Status == types.Complete || msg.Status == types.Failed || msg.Status == types.Filtered {
			delete(state, msg.IrisLookupId)
			store[msg.IrisLookupId] = *msg
		}
	}
}

func init() {
	cobra.OnInitialize(func() {})
}

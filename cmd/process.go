package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cosmossdk.io/log"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/circle"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
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

// SequenceMap maps the domain -> the equivalent minter account sequence or nonce
var sequenceMap = types.NewSequenceMap()

func Start(cmd *cobra.Command, args []string) {
	// messageState processing queue
	var processingQueue = make(chan *types.TxState, 10000)

	sigTerm := make(chan os.Signal, 1)

	registeredDomains := make(map[types.Domain]types.Chain)

	for name, cfg := range Cfg.Chains {
		c, err := cfg.Chain(name)
		if err != nil {
			Logger.Error("Error creating chain", "err: ", err)
			os.Exit(1)
		}

		if err := c.InitializeBroadcaster(cmd.Context(), Logger, sequenceMap); err != nil {
			Logger.Error("Error initializing broadcaster", "err: ", err)
			os.Exit(1)
		}

		go c.StartListener(cmd.Context(), Logger, processingQueue, sigTerm)

		if _, ok := registeredDomains[c.Domain()]; ok {
			Logger.Error("Duplicate domain found", "domain", c.Domain())
			os.Exit(1)
		}

		registeredDomains[c.Domain()] = c
	}

	// spin up Processor worker pool
	for i := 0; i < int(Cfg.ProcessorWorkerCount); i++ {
		go StartProcessor(cmd.Context(), Cfg, Logger, registeredDomains, processingQueue, sequenceMap)
	}

	signal.Notify(sigTerm, os.Interrupt, syscall.SIGTERM)
	<-sigTerm
}

// StartProcessor is the main processing pipeline.
func StartProcessor(
	ctx context.Context,
	cfg types.Config,
	logger log.Logger,
	registeredDomains map[types.Domain]types.Chain,
	processingQueue chan *types.TxState,
	sequenceMap *types.SequenceMap,
) {
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
				filterInvalidDestinationCallers(registeredDomains, logger, msg) {
				msg.Status = types.Filtered
			}

			// if the message is burned or pending, check for an attestation
			if msg.Status == types.Created || msg.Status == types.Pending {
				response := circle.CheckAttestation(cfg.Circle.AttestationBaseUrl, logger, msg.IrisLookupId, msg.SourceTxHash, msg.SourceDomain, msg.DestDomain)
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
			chain, ok := registeredDomains[domain]
			if !ok {
				logger.Error("No chain registered for domain", "domain", domain)
				continue
			}

			if err := chain.Broadcast(ctx, logger, msgs, sequenceMap); err != nil {
				logger.Error("unable to mint one or more transfers", "error(s)", err, "total_transfers", len(msgs), "name", chain.Name(), "domain", domain)
				requeue = true
				continue
			}

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
func filterDisabledCCTPRoutes(cfg types.Config, logger log.Logger, msg *types.MessageState) bool {
	val, ok := cfg.EnabledRoutes[msg.SourceDomain]
	result := !(ok && val == msg.DestDomain)
	if result {
		logger.Info(fmt.Sprintf("Filtered tx %s because relaying from %d to %d is not enabled",
			msg.SourceTxHash, msg.SourceDomain, msg.DestDomain))
	}
	return result
}

// filterInvalidDestinationCallers returns true if the minter is not the destination caller for the specified domain
func filterInvalidDestinationCallers(registeredDomains map[types.Domain]types.Chain, logger log.Logger, msg *types.MessageState) bool {
	chain, ok := registeredDomains[msg.DestDomain]
	if !ok {
		logger.Error("No chain registered for domain", "domain", msg.DestDomain)
		return true
	}

	return !chain.IsDestinationCaller(msg.DestinationCaller)
}

func LookupKey(sourceTxHash string) string {
	// return fmt.Sprintf("%s-%s", sourceTxHash, messageType)
	return sourceTxHash
}

func init() {
	cobra.OnInitialize(func() {})
}

package ethereum

import (
	"cosmossdk.io/log"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"math/big"
	"time"
)

// Broadcast broadcasts a message to Ethereum
func Broadcast(
	cfg config.Config,
	logger log.Logger,
	msg *types.MessageState,
	sequenceMap *types.SequenceMap,
) (*ethtypes.Transaction, error) {

	// set up eth client
	client, err := ethclient.Dial(cfg.Networks.Destination.Ethereum.RPC)
	defer client.Close()

	privEcdsaKey, ethereumAddress, err := GetEcdsaKeyAddress(cfg.Networks.Minters[0].MinterPrivateKey)
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privEcdsaKey, big.NewInt(cfg.Networks.Destination.Ethereum.ChainId))
	messageTransmitter, err := NewMessageTransmitter(common.HexToAddress(cfg.Networks.Source.Ethereum.MessageTransmitter), client)
	attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}

	for attempt := 0; attempt <= cfg.Networks.Destination.Ethereum.BroadcastRetries; attempt++ {
		logger.Info(fmt.Sprintf(
			"Broadcasting %s message from %d to %d: with source tx hash %s",
			msg.Type,
			msg.SourceDomain,
			msg.DestDomain,
			msg.SourceTxHash))

		// TODO Account sequence lock is implemented but gets out of sync with remote.
		// accountSequence := sequenceMap.Next(cfg.Networks.Destination.Noble.DomainId)
		_, err := GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, ethereumAddress)
		if err != nil {
			logger.Error("unable to retrieve ethereum account nonce")
			continue
		}
		//auth.Nonce = big.NewInt(nonce)

		// broadcast txn
		tx, err := messageTransmitter.ReceiveMessage(
			auth,
			msg.MsgSentBytes,
			attestationBytes,
		)
		if err == nil {
			msg.Status = types.Complete
			return tx, nil
		} else {

			logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
			if parsedErr, ok := err.(JsonError); ok {
				if parsedErr.ErrorCode() == 3 && parsedErr.Error() == "execution reverted: Nonce already used" {
					msg.Status = types.Failed
					return nil, errors.New(fmt.Sprintf("Nonce already used"))
				}
			}

			// if it's not the last attempt, retry
			if attempt != cfg.Networks.Destination.Ethereum.BroadcastRetries {
				logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Ethereum.BroadcastRetryInterval))
				time.Sleep(time.Duration(cfg.Networks.Destination.Ethereum.BroadcastRetryInterval) * time.Second)
			}
			continue
		}
	}
	msg.Status = types.Failed

	return nil, errors.New("reached max number of broadcast attempts")
}

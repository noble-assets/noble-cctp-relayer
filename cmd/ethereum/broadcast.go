package ethereum

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// Broadcast broadcasts a message to Ethereum
func Broadcast(
	ctx context.Context,
	cfg config.Config,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
) ([]*ethtypes.Transaction, error) {

	// set up eth client
	client, err := ethclient.Dial(cfg.Networks.Destination.Ethereum.RPC)
	if err != nil {
		return nil, fmt.Errorf("unable to dial ethereum client: %w", err)
	}
	defer client.Close()

	backend := NewContractBackendWrapper(client)

	privEcdsaKey, ethereumAddress, err := GetEcdsaKeyAddress(cfg.Networks.Minters[0].MinterPrivateKey)
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privEcdsaKey, big.NewInt(cfg.Networks.Destination.Ethereum.ChainId))
	if err != nil {
		return nil, fmt.Errorf("unable to create auth: %w", err)
	}

	messageTransmitter, err := NewMessageTransmitter(common.HexToAddress(cfg.Networks.Source.Ethereum.MessageTransmitter), backend)
	if err != nil {
		return nil, fmt.Errorf("unable to create message transmitter: %w", err)
	}

	var broadcastErrors error
	var txs []*ethtypes.Transaction
	for _, msg := range msgs {

		attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
		if err != nil {
			return nil, errors.New("unable to decode message attestation")
		}

		for attempt := 0; attempt <= cfg.Networks.Destination.Ethereum.BroadcastRetries; attempt++ {
			logger.Info(fmt.Sprintf(
				"Broadcasting message from %d to %d: with source tx hash %s",
				msg.SourceDomain,
				msg.DestDomain,
				msg.SourceTxHash))

			nonce := sequenceMap.Next(cfg.Networks.Destination.Ethereum.DomainId)
			auth.Nonce = big.NewInt(nonce)

			// TODO remove
			nextNonce, err := GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, ethereumAddress)
			if err != nil {
				logger.Error("unable to retrieve account number")
			} else {
				auth.Nonce = big.NewInt(nextNonce)
			}
			// TODO end remove

			// check if nonce already used
			co := &bind.CallOpts{
				Pending: true,
				Context: ctx,
			}

			logger.Debug("Checking if nonce was used for broadcast to Ethereum", "source_domain", msg.SourceDomain, "nonce", msg.Nonce)

			key := append(
				common.LeftPadBytes((big.NewInt(int64(msg.SourceDomain))).Bytes(), 4),
				common.LeftPadBytes((big.NewInt(int64(msg.Nonce))).Bytes(), 8)...,
			)

			response, nonceErr := messageTransmitter.UsedNonces(co, [32]byte(crypto.Keccak256(key)))
			if nonceErr != nil {
				logger.Debug("Error querying whether nonce was used.   Continuing...")
			} else {
				fmt.Printf("received used nonce response: %d\n", response)
				if response.Uint64() == uint64(1) {
					// nonce has already been used, mark as complete
					logger.Debug(fmt.Sprintf("This source domain/nonce has already been used: %d %d",
						msg.SourceDomain, msg.Nonce))
					msg.Status = types.Complete
					return nil, errors.New("receive message was already broadcasted")
				}
			}

			// broadcast txn
			tx, err := messageTransmitter.ReceiveMessage(
				auth,
				msg.MsgSentBytes,
				attestationBytes,
			)
			if err == nil {
				msg.Status = types.Complete
				txs = append(txs, tx)
				continue
			} else {
				logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
				if parsedErr, ok := err.(JsonError); ok {
					if parsedErr.ErrorCode() == 3 && parsedErr.Error() == "execution reverted: Nonce already used" {
						msg.Status = types.Complete
						return nil, parsedErr
					}

					match, _ := regexp.MatchString("nonce too low: next nonce [0-9]+, tx nonce [0-9]+", parsedErr.Error())
					if match {
						numberRegex := regexp.MustCompile("[0-9]+")
						nextNonce, err := strconv.ParseInt(numberRegex.FindAllString(parsedErr.Error(), 1)[0], 10, 0)
						if err != nil {
							nextNonce, err = GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, ethereumAddress)
							if err != nil {
								logger.Error("unable to retrieve account number")
							}
						}
						sequenceMap.Put(cfg.Networks.Destination.Ethereum.DomainId, nextNonce)
					}
				}

				// if it's not the last attempt, retry
				// TODO increase the destination.ethereum.broadcast retries (3-5) and retry interval (15s).  By checking for used nonces, there is no gas cost for failed mints.
				if attempt != cfg.Networks.Destination.Ethereum.BroadcastRetries {
					logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Ethereum.BroadcastRetryInterval))
					time.Sleep(time.Duration(cfg.Networks.Destination.Ethereum.BroadcastRetryInterval) * time.Second)
				}
				continue
			}
		}
		msg.Status = types.Failed

		broadcastErrors = errors.Join(broadcastErrors, errors.New("reached max number of broadcast attempts"))
	}
	return nil, broadcastErrors
}

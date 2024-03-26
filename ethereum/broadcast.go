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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/noble-cctp-relayer/ethereum/contracts"
	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

func (e *Ethereum) InitializeBroadcaster(
	ctx context.Context,
	logger log.Logger,
	sequenceMap *types.SequenceMap,
) error {
	nextNonce, err := GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
	if err != nil {
		return fmt.Errorf("unable to retrieve evm account nonce: %w", err)
	}
	sequenceMap.Put(e.Domain(), uint64(nextNonce))

	return nil
}

func (e *Ethereum) Broadcast(
	ctx context.Context,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
	m *relayer.PromMetrics,
) error {

	logger = logger.With("chain", e.name, "chain_id", e.chainID, "domain", e.domain)

	backend := NewContractBackendWrapper(e.rpcClient)

	auth, err := bind.NewKeyedTransactorWithChainID(e.privateKey, big.NewInt(e.chainID))
	if err != nil {
		return fmt.Errorf("unable to create auth: %w", err)
	}

	messageTransmitter, err := contracts.NewMessageTransmitter(common.HexToAddress(e.messageTransmitterAddress), backend)
	if err != nil {
		return fmt.Errorf("unable to create message transmitter: %w", err)
	}

	var broadcastErrors error
MsgLoop:
	for _, msg := range msgs {
		attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
		if err != nil {
			return errors.New("unable to decode message attestation")
		}

		for attempt := 0; attempt <= e.maxRetries; attempt++ {
			// check if another worker already broadcasted tx due to flush
			if msg.Status == types.Complete {
				continue MsgLoop
			}

			if err := e.attemptBroadcast(
				ctx,
				logger,
				msg,
				sequenceMap,
				auth,
				messageTransmitter,
				attestationBytes,
			); err == nil {
				continue MsgLoop
			}

			// if it's not the last attempt, retry
			// TODO increase the destination.ethereum.broadcast retries (3-5) and retry interval (15s).  By checking for used nonces, there is no gas cost for failed mints.
			if attempt != e.maxRetries {
				logger.Info(fmt.Sprintf("Retrying in %d seconds", e.retryIntervalSeconds))
				time.Sleep(time.Duration(e.retryIntervalSeconds) * time.Second)
			}
		}

		// retried max times with failure
		if m != nil {
			m.IncBroadcastErrors(e.name, fmt.Sprint(e.domain))
		}
		broadcastErrors = errors.Join(broadcastErrors, errors.New("reached max number of broadcast attempts"))
	}
	return broadcastErrors
}

func (e *Ethereum) attemptBroadcast(
	ctx context.Context,
	logger log.Logger,
	msg *types.MessageState,
	sequenceMap *types.SequenceMap,
	auth *bind.TransactOpts,
	messageTransmitter *contracts.MessageTransmitter,
	attestationBytes []byte,
) error {
	logger.Info(fmt.Sprintf(
		"Broadcasting message from %d to %d: with source tx hash %s",
		msg.SourceDomain,
		msg.DestDomain,
		msg.SourceTxHash))

	nonce := sequenceMap.Next(e.domain)
	auth.Nonce = big.NewInt(int64(nonce))

	e.mu.Lock()
	defer e.mu.Unlock()

	// TODO remove
	nextNonce, err := GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
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
		logger.Debug("Error querying whether nonce was used.   Continuing...", "error:", nonceErr)
	} else {
		if response.Uint64() == uint64(1) {
			// nonce has already been used, mark as complete
			logger.Debug(fmt.Sprintf("This source domain/nonce has already been used: %d %d",
				msg.SourceDomain, msg.Nonce), "src-tx", msg.SourceTxHash, "reviever")
			msg.Status = types.Complete
			return nil
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

		msg.DestTxHash = tx.Hash().Hex()

		logger.Info(fmt.Sprintf("Successfully broadcast %s to Ethereum.  Tx hash: %s", msg.SourceTxHash, msg.DestTxHash))

		return nil
	}

	logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
	if parsedErr, ok := err.(JsonError); ok {
		if parsedErr.ErrorCode() == 3 && parsedErr.Error() == "execution reverted: Nonce already used" {
			msg.Status = types.Complete
			logger.Error(fmt.Sprintf("This account nonce has already been used: %d", nonce))

			return nil
		}

		match, _ := regexp.MatchString("nonce too low: next nonce [0-9]+, tx nonce [0-9]+", parsedErr.Error())
		if match {
			numberRegex := regexp.MustCompile("[0-9]+")
			nextNonce, err := strconv.ParseInt(numberRegex.FindAllString(parsedErr.Error(), 1)[0], 10, 0)
			if err != nil {
				nextNonce, err = GetEthereumAccountNonce(e.rpcURL, e.minterAddress)
				if err != nil {
					logger.Error("unable to retrieve account number")
				}
			}
			sequenceMap.Put(e.domain, uint64(nextNonce))
		}
	}

	return err
}

package ethereum

import (
	"cosmossdk.io/log"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd"
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
) (*types.Transaction, error) {
	// build txn
	attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}

	// get priv key
	ethereumAddress := cfg.Networks.Minters[0].MinterAddress

	// sign tx
	addr, _ := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	if addr != ethereumAddress {
		return nil, fmt.Errorf("private key (%s) does not match noble address (%s)", addr, ethereumAddress)
	}

	// set up eth client
	client, err := ethclient.Dial(cfg.Networks.Destination.Ethereum.RPC)
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(cfg.Networks.Minters[0].MinterPrivateKey)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(5))

	messageTransmitter, err := cmd.NewMessageTransmitter(common.HexToAddress("MessageTransmitterAddress"), client) // TODO address

	for attempt := 0; attempt <= cfg.Networks.Destination.Ethereum.BroadcastRetries; attempt++ {
		logger.Debug(fmt.Sprintf(
			"Broadcasting %s message from %d to %d: with source tx hash %s",
			msg.Type,
			msg.SourceDomain,
			msg.DestDomain,
			msg.SourceTxHash))

		// TODO Account sequence lock is implemented but gets out of sync with remote.
		// accountSequence := sequenceMap.Next(cfg.Networks.Destination.Noble.DomainId)
		nonce, err := GetEthereumAccountNonce(cfg.Networks.Destination.Ethereum.RPC, ethereumAddress)
		if err != nil {
			logger.Error("unable to retrieve ethereum account nonce")
		}

		// broadcast txn
		// broadcast txn
		tx, err := messageTransmitter.ReceiveMessage(
			auth,
			[]byte{},
			[]byte{},
		)
		if err == nil {
			msg.Status = types.Complete
			return tx, nil
		} else {
			logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
			logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Ethereum.BroadcastRetryInterval))
			time.Sleep(time.Duration(cfg.Networks.Destination.Ethereum.BroadcastRetryInterval) * time.Second)
			continue
		}
	}
	msg.Status = types.Failed

	return nil, errors.New("reached max number of broadcast attempts")
}

// TODO
func GetEthereumAccountNonce(urlBase string, address string) (int64, error) {
	return 0, nil
}

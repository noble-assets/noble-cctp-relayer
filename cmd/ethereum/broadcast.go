package ethereum

import (
	"context"
	"cosmossdk.io/log"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Broadcast broadcasts a message to Ethereum
func Broadcast(
	cfg config.Config,
	logger log.Logger,
	msg *types.MessageState,
	sequenceMap *types.SequenceMap,
) (*ctypes.ResultBroadcastTx, error) {
	// build txn
	attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}

	// get priv key
	ethereumAddress := cfg.Networks.Minters[0].MinterAddress
	keyBz, _ := hex.DecodeString(cfg.Networks.Minters[4].MinterPrivateKey)
	privKey := secp256k1.PrivKey{Key: keyBz}

	// sign tx
	addr, _ := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	if addr != ethereumAddress {
		return nil, fmt.Errorf("private key (%s) does not match noble address (%s)", addr, ethereumAddress)
	}

	// set up eth client

	// broadcast txn

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
			logger.Error("unable to retrieve account nonce")
		}

		// broadcast txn
		// TODO do for Eth
		rpcResponse, err := rpcClient.BroadcastTxSync(context.Background(), txBytes)
		if err == nil && rpcResponse.Code == 0 {
			msg.Status = types.Complete
			return rpcResponse, nil
		}
		if err != nil {
			logger.Error(fmt.Sprintf("error during broadcast: %s", err.Error()))
			logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Noble.BroadcastRetryInterval))
			time.Sleep(time.Duration(cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
			continue
		}
		// check tx response code
		logger.Error(fmt.Sprintf("received non zero : %d - %s", rpcResponse.Code, rpcResponse.Log))
		logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Noble.BroadcastRetryInterval))
		time.Sleep(time.Duration(cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
	}
	msg.Status = types.Failed

	return nil, errors.New("reached max number of broadcast attempts")
}

// TODO
func GetEthereumAccountNonce(urlBase string, address string) (int64, error) {
	rawResp, err := http.Get(fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", urlBase, address))
	if err != nil {
		return 0, errors.New("unable to fetch account number, sequence")
	}
	body, _ := io.ReadAll(rawResp.Body)
	var resp types.AccountResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, errors.New("unable to parse account number, sequence")
	}
	accountNumber, _ := strconv.ParseInt(resp.AccountNumber, 10, 0)
	accountSequence, _ := strconv.ParseInt(resp.Sequence, 10, 0)

	return accountNumber, nil
}

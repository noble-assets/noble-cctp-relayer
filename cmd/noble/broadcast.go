package noble

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"cosmossdk.io/log"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	libclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// Broadcast broadcasts a message to Noble
func Broadcast(
	ctx context.Context,
	cfg config.Config,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
) error {
	// set up sdk context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	sdkContext := sdkClient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()

	// get priv key
	nobleAddress := cfg.Networks.Minters[4].MinterAddress
	keyBz, err := hex.DecodeString(cfg.Networks.Minters[4].MinterPrivateKey)
	if err != nil {
		return fmt.Errorf("unable to parse Noble private key")
	}
	privKey := secp256k1.PrivKey{Key: keyBz}

	rpcClient, err := NewRPCClient(cfg.Networks.Destination.Noble.RPC, 10*time.Second)
	if err != nil {
		return errors.New("failed to set up rpc client")
	}

	cc, err := cosmos.NewProvider(cfg.Networks.Source.Noble.RPC)
	if err != nil {
		return fmt.Errorf("unable to build cosmos provider for noble: %w", err)
	}

	// sign and broadcast txn
	for attempt := 0; attempt <= cfg.Networks.Destination.Noble.BroadcastRetries; attempt++ {

		var receiveMsgs []sdk.Msg
		for _, msg := range msgs {

			used, err := cc.QueryUsedNonce(ctx, types.Domain(msg.SourceDomain), msg.Nonce)
			if err != nil {
				return fmt.Errorf("unable to query used nonce: %w", err)
			}

			if used {
				msg.Status = types.Complete
				logger.Info(fmt.Sprintf("Noble cctp minter nonce %d already used", msg.Nonce))
				continue
			}

			attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
			if err != nil {
				return fmt.Errorf("unable to decode message attestation")
			}

			receiveMsgs = append(receiveMsgs, nobletypes.NewMsgReceiveMessage(
				nobleAddress,
				msg.MsgSentBytes,
				attestationBytes,
			))

			logger.Info(fmt.Sprintf(
				"Broadcasting message from %d to %d: with source tx hash %s",
				msg.SourceDomain,
				msg.DestDomain,
				msg.SourceTxHash))
		}

		err = txBuilder.SetMsgs(receiveMsgs...)
		if err != nil {
			return fmt.Errorf("failed to set messages on tx: %w", err)
		}

		txBuilder.SetGasLimit(cfg.Networks.Destination.Noble.GasLimit)
		// TODO: make configurable
		txBuilder.SetMemo("Thank you for relaying with Strangelove")

		accountSequence := sequenceMap.Next(cfg.Networks.Destination.Noble.DomainId)
		//TODO: don't need to fetch this everytime
		accountNumber, _, err := GetNobleAccountNumberSequence(cfg.Networks.Destination.Noble.API, nobleAddress)

		if err != nil {
			return fmt.Errorf("failed to retrieve account number and sequence: %w", err)
		}

		sigV2 := signing.SignatureV2{
			PubKey: privKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: uint64(accountSequence),
		}

		signerData := xauthsigning.SignerData{
			ChainID:       cfg.Networks.Destination.Noble.ChainId,
			AccountNumber: uint64(accountNumber),
			Sequence:      uint64(accountSequence),
		}

		txBuilder.SetSignatures(sigV2)

		sigV2, err = clientTx.SignWithPrivKey(
			sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			signerData,
			txBuilder,
			&privKey,
			sdkContext.TxConfig,
			uint64(accountSequence),
		)
		if err != nil {
			return fmt.Errorf("failed to sign tx: %w", err)
		}

		if err := txBuilder.SetSignatures(sigV2); err != nil {
			return fmt.Errorf("failed to set signatures: %w", err)
		}

		// Generated Protobuf-encoded bytes.
		txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return fmt.Errorf("failed to proto encode tx: %w", err)
		}

		rpcResponse, err := rpcClient.BroadcastTxSync(context.Background(), txBytes)
		if err != nil || (rpcResponse != nil && rpcResponse.Code != 0) {
			// Log the error
			logger.Error(fmt.Sprintf("error during broadcast: %s", getErrorString(err, rpcResponse)))

			if err != nil || rpcResponse == nil {
				// Log retry information
				logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Noble.BroadcastRetryInterval))
				time.Sleep(time.Duration(cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
				// wait a random amount of time to lower probability of concurrent message nonce collision
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
				continue
			}

			// Log details for non-zero response code
			logger.Error(fmt.Sprintf("received non-zero: %d - %s", rpcResponse.Code, rpcResponse.Log))

			// Handle specific error code (32)
			if rpcResponse.Code == 32 {
				newAccountSequence := extractAccountSequence(logger, rpcResponse.Log, nobleAddress, cfg.Networks.Destination.Noble.API)
				logger.Debug(fmt.Sprintf("retrying with new account sequence: %d", newAccountSequence))
				sequenceMap.Put(cfg.Networks.Destination.Noble.DomainId, newAccountSequence)
			}

			// Log retry information
			logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Noble.BroadcastRetryInterval))
			time.Sleep(time.Duration(cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
			// wait a random amount of time to lower probability of concurrent message nonce collision
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			continue
		}

		// Tx was successfully broadcast
		for _, msg := range msgs {
			msg.DestTxHash = rpcResponse.Hash.String()
			msg.Status = types.Complete
		}
		logger.Info(fmt.Sprintf("Successfully broadcast %s to Noble.  Tx hash: %s", msgs[0].SourceTxHash, msgs[0].DestTxHash))

		return nil
	}

	for _, msg := range msgs {
		if msg.Status != types.Complete {
			msg.Status = types.Failed
		}
	}

	return errors.New("reached max number of broadcast attempts")
}

// getErrorString returns the appropriate value to log when tx broadcast errors are encountered.
func getErrorString(err error, rpcResponse *ctypes.ResultBroadcastTx) string {
	if rpcResponse != nil {
		return rpcResponse.Log
	}
	return err.Error()
}

// extractAccountSequence attempts to extract the account sequence number from the RPC response logs when
// account sequence mismatch errors are encountered. If the account sequence number cannot be extracted from the logs,
// it is retrieved by making a request to the API endpoint.
func extractAccountSequence(logger log.Logger, rpcResponseLog, nobleAddress, nobleAPI string) int64 {
	pattern := `expected (\d+), got (\d+)`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(rpcResponseLog)

	if len(match) == 3 {
		// Extract the numbers from the match.
		newAccountSequence, _ := strconv.ParseInt(match[1], 10, 0)
		return newAccountSequence
	}

	// Otherwise, just request the account sequence
	_, newAccountSequence, err := GetNobleAccountNumberSequence(nobleAPI, nobleAddress)
	if err != nil {
		logger.Error("unable to retrieve account number")
	}

	return newAccountSequence
}

// NewRPCClient initializes a new tendermint RPC client connected to the specified address.
func NewRPCClient(addr string, timeout time.Duration) (*rpchttp.HTTP, error) {
	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = timeout
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}
	return rpcClient, nil
}

func GetNobleAccountNumberSequence(urlBase string, address string) (int64, int64, error) {
	rawResp, err := http.Get(fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", urlBase, address))
	if err != nil {
		return 0, 0, errors.New("unable to fetch account number, sequence")
	}
	body, _ := io.ReadAll(rawResp.Body)
	var resp types.AccountResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to parse account number, sequence. Raw HHTP Get response: %s", string(body))
	}
	accountNumber, _ := strconv.ParseInt(resp.AccountNumber, 10, 0)
	accountSequence, _ := strconv.ParseInt(resp.Sequence, 10, 0)

	return accountNumber, accountSequence, nil
}

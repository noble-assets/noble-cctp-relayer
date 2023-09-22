package noble

import (
	"context"
	"strconv"

	//authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// Broadcast broadcasts a message to Noble
func Broadcast(
	cfg config.Config,
	logger log.Logger,
	msg *types.MessageState,
) (*ctypes.ResultBroadcastTx, error) {
	// set up sdk context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	sdkContext := sdkClient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()
	attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}

	// get priv key
	nobleAddress := cfg.Networks.Minters[4].MinterAddress
	keyBz, _ := hex.DecodeString(cfg.Networks.Minters[4].MinterPrivateKey)
	privKey := secp256k1.PrivKey{Key: keyBz}

	if err != nil {
		return nil, errors.New("unable to convert priv key to noble address")
	}

	receiveMsg := nobletypes.NewMsgReceiveMessage(
		nobleAddress,
		msg.MsgSentBytes,
		attestationBytes,
	)
	err = txBuilder.SetMsgs(receiveMsg)
	if err != nil {
		return nil, err
	}

	txBuilder.SetGasLimit(cfg.Networks.Destination.Noble.GasLimit)
	txBuilder.SetMemo("Thank you for relaying with Strangelove")

	// sign tx

	// get account number, sequence
	fmt.Println(fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", cfg.Networks.Destination.Noble.API, nobleAddress))
	rawResp, err := http.Get(fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", cfg.Networks.Destination.Noble.API, nobleAddress))
	if err != nil {
		return nil, errors.New("unable to fetch account number, sequence")
	}
	body, _ := io.ReadAll(rawResp.Body)
	var resp types.AccountResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, errors.New("unable to parse account number, sequence")
	}
	accountNumber, _ := strconv.ParseInt(resp.AccountNumber, 10, 0)
	accountSequence, _ := strconv.ParseInt(resp.Sequence, 10, 0)

	addr, _ := bech32.ConvertAndEncode("noble", privKey.PubKey().Address())
	if addr != nobleAddress {
		return nil, fmt.Errorf("private key (%s) does not match noble address (%s)", addr, nobleAddress)
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

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	// Generated Protobuf-encoded bytes.
	txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	// broadcast txn

	// set up client
	rpcClient, err := NewRPCClient(cfg.Networks.Destination.Noble.RPC, 10*time.Second)
	if err != nil {
		return nil, errors.New("failed to set up rpc client")
	}

	for attempt := 0; attempt <= cfg.Networks.Destination.Noble.BroadcastRetries+1; attempt++ {
		logger.Info(fmt.Sprintf(
			"Broadcasting message from %d to %d: with source tx hash %s",
			msg.SourceDomain,
			msg.DestDomain,
			msg.SourceTxHash))

		// TODO change to commit
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

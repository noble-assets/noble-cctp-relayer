package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	"github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/strangelove-ventures/noble-cctp-relayer/utils"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"os"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions between Ethereum and Noble",
	Run:   Start,
}

var pendingAttestations chan types.Attestation

func Start(cmd *cobra.Command, args []string) {
	pendingAttestations = make(chan types.Attestation, 1000) // TODO
	client, err := ethclient.Dial(Cfg.Networks.Ethereum.RPC)
	if err != nil {
		logger.Error("unable to initialise ethereum client", "err", err)
		os.Exit(1)
	}

	messages := make(chan ethtypes.Log)

	filter := ethereum.FilterQuery{
		Addresses: []common.Address{MessageTransmitter},
		Topics:    [][]common.Hash{{MessageSent.ID}},
	}

	sub, err := client.SubscribeFilterLogs(context.Background(), filter, messages)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	// TODO remove after testing
	go func() {
		utils.InjectMessages(client, filter, messages)
	}()

	for {
		select {
		case err := <-sub.Err():
			logger.Error("connection closed", "err", err)
			os.Exit(1)
		case msg := <-messages:
			HandleMessage(msg)
		}
	}
}

func HandleMessage(log ethtypes.Log) {
	event := make(map[string]interface{})
	_ = MessageTransmitterABI.UnpackIntoMap(event, MessageSent.Name, log.Data)

	rawMessage := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessage)

	if message.DestinationDomain != Cfg.Networks.Noble.DomainId {
		logger.Debug("received irrelevant message", "destination", message.DestinationDomain, "tx", log.TxHash)
		return
	}

	if _, err := new(types.BurnMessage).Parse(message.MessageBody); err == nil {
		logger.Info("received a new burn message", "nonce", message.Nonce, "tx", log.TxHash)

		pendingAttestations <- types.Attestation{
			Message: message.MessageBody,
			Key:     hex.EncodeToString(crypto.Keccak256(message.MessageBody)),
		}

		return
	}

	if content, err := new(types.MetadataMessage).Parse(message.MessageBody); err == nil {
		logger.Info("received a new forward message", "channel", content.Channel, "tx", log.TxHash)

		pendingAttestations <- types.Attestation{
			Message: message.MessageBody,
			Key:     hex.EncodeToString(crypto.Keccak256(message.MessageBody)),
		}
		return
	}
}

func Receive() {
	for {
		select {
		case attestation := <-pendingAttestations:
			// goroutine
			if AttestationIsReady(&attestation) {
				Mint(attestation)
			}
		}
	}
}

// AttestationIsReady checks the iris api for attestation status
// returns true if attestation is complete
func AttestationIsReady(attestation *types.Attestation) bool {
	rawResponse, err := http.Get(Cfg.AttestationBaseUrl + "0x" + attestation.Key)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		//logger.Info("non 200 response received")
		return false
	}
	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		logger.Debug("unable to parse message body")
		return false
	}

	response := types.AttestationResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil || response.Status != "complete" {
		logger.Info("unable to unmarshal response")
		return false
	}

	attestation.Attestation = response.Attestation

	return true
}

func Mint(attestation types.Attestation) error {

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	client := sdkClient.Context{
		ChainID:  "noble-1",
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
		//AccountRetriever: nil,
		//NodeURI:          "",
		//Codec: cdc,
	}
	msg := nobletypes.NewMsgReceiveMessage(
		"",
		attestation.Message,
		[]byte(attestation.Attestation),
	)

	txBuilder := client.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return err
	}

	//txBuilder.SetGasLimit(1)
	//txBuilder.SetFeeAmount(1)
	//txBuilder.SetMemo("")
	//txBuilder.SetTimeoutHeight(1)

	// sign tx
	priv, _, _ := testdata.KeyTestPubAddr()
	accNumber := uint64(0)
	accSeq := uint64(0)

	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  client.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: accSeq,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return err
	}

	signerData := xauthsigning.SignerData{
		ChainID:       Cfg.Networks.Noble.ChainId,
		AccountNumber: accNumber,
		Sequence:      accSeq,
	}

	sigV2, err = clientTx.SignWithPrivKey(
		client.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		priv,
		client.TxConfig,
		accSeq,
	)

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return err
	}

	// Generated Protobuf-encoded bytes.
	txBytes, err := client.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return err
	}

	// BROADCAST

	// set up grpc client
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	txClient := tx.NewServiceClient(grpcConn)
	grpcRes, err := txClient.BroadcastTx(
		context.Background(),
		&tx.BroadcastTxRequest{
			TxBytes: txBytes,
			Mode:    0,
		},
	)
	if err != nil {
		return err
	}

	fmt.Println(grpcRes.TxResponse.Code) // should be 0

	return nil
}

func init() {
	cobra.OnInitialize(func() {
		messageTransmitter, err := os.Open("./abi/MessageTransmitter.json")
		if err != nil {
			logger.Error("unable to read MessageTransmitter abi", "err", err)
			os.Exit(1)
		}
		MessageTransmitterABI, err = abi.JSON(messageTransmitter)
		if err != nil {
			logger.Error("unable to parse MessageTransmitter abi", "err", err)
		}

		MessageSent = MessageTransmitterABI.Events["MessageSent"]
	})
}

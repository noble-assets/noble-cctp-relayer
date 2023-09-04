package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	xauthtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kr/pretty"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"google.golang.org/grpc"
	"io"
	"math/big"
	"net/http"
	"os"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions between Ethereum and Noble",
	Run:   Start,
}

func Start(cmd *cobra.Command, args []string) {
	ethClient, err := ethclient.Dial(Cfg.Networks.Ethereum.RPC)
	if err != nil {
		logger.Error("unable to initialize ethereum ethClient", "err", err)
		os.Exit(1)
	}

	// receive eth events and pass to logChan
	Run(ethClient)

}

// websockets do not work
// https://github.com/ethereum/go-ethereum/issues/15063
func Run(ethClient *ethclient.Client) {
	logs := make(chan ethtypes.Log)

	MessageTransmitter = common.HexToAddress(Cfg.Networks.Ethereum.MessageTransmitter)
	MessageTransmitter = common.HexToAddress("0x147B8eb97fD247D06C4006D269c90C1908Fb5D54") // todo del

	query := ethereum.FilterQuery{
		//Addresses: []common.Address{MessageTransmitter},
		//Topics:    [][]common.Hash{{MessageSent.ID}},
		FromBlock: big.NewInt(9637317),
		ToBlock:   big.NewInt(9637335),
	}

	sub, err := ethClient.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	for {
		select {
		case err := <-sub.Err():
			logger.Error("connection closed", "err", err)
			os.Exit(1)
		case log := <-logs:
			attestation := ProcessLog(log)

			if attestation != nil && CheckAttestation(attestation) {
				Mint(attestation)

			}
		}

	}
}

func ProcessLog(log ethtypes.Log) *types.Attestation {
	logger.Info("ProcessLog")
	event := make(map[string]interface{})
	_ = MessageTransmitterABI.UnpackIntoMap(event, MessageSent.Name, log.Data)

	rawMessage := event["message"].([]byte)
	message, _ := new(types.Message).Parse(rawMessage)

	if message.DestinationDomain != Cfg.Networks.Noble.DomainId {
		logger.Debug("received irrelevant message", "destination", message.DestinationDomain, "tx", log.TxHash)
		return nil
	}

	if _, err := new(types.BurnMessage).Parse(message.MessageBody); err == nil {
		logger.Info("received a new burn message", "nonce", message.Nonce, "tx", log.TxHash)

		return &types.Attestation{
			Message: message.MessageBody,
			Key:     hex.EncodeToString(crypto.Keccak256(message.MessageBody)),
		}
	}

	if content, err := new(types.MetadataMessage).Parse(message.MessageBody); err == nil {
		logger.Info("received a new forward message", "channel", content.Channel, "tx", log.TxHash)

		return &types.Attestation{
			Message: message.MessageBody,
			Key:     hex.EncodeToString(crypto.Keccak256(message.MessageBody)),
		}
	}

	logger.Info(pretty.Sprintf("unable to parse txn into message.  tx hash %s"))
	return nil
}

// CheckAttestation checks the iris api for attestation status
// returns true if attestation is complete
func CheckAttestation(attestation *types.Attestation) bool {
	logger.Info("CheckAttestation for " + Cfg.AttestationBaseUrl + "0x" + attestation.Key)
	rawResponse, err := http.Get(Cfg.AttestationBaseUrl + "0x" + attestation.Key)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		logger.Debug("non 200 response received")
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
		logger.Debug("unable to unmarshal response")
		return false
	}

	attestation.Attestation = response.Attestation

	return true
}

func Mint(attestation *types.Attestation) error {
	logger.Info("Mint")

	// set up sdk context
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sdkContext := sdkClient.Context{
		ChainID:          Cfg.Networks.Noble.ChainId,
		TxConfig:         xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
		AccountRetriever: xauthtypes.AccountRetriever{},
		//NodeURI:          "",
		//Codec: cdc,
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()
	msg := nobletypes.NewMsgReceiveMessage(
		"", // TODO
		attestation.Message,
		[]byte(attestation.Attestation),
	)
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return err
	}

	//txBuilder.SetGasLimit(1)
	//txBuilder.SetFeeAmount(1)
	txBuilder.SetMemo("Thank you for relaying with Strangelove")
	//txBuilder.SetTimeoutHeight(1)

	// sign tx
	priv, _, _ := testdata.KeyTestPubAddr() // TODO delete

	// get account number, sequence
	addrBytes, err := sdktypes.GetFromBech32(Cfg.Networks.Noble.MinterAddress, "noble")
	if err != nil {
		return err
	}
	accountNumber, accountSequence, err := sdkContext.AccountRetriever.GetAccountNumberSequence(sdkContext, addrBytes)

	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: accountSequence,
	}

	signerData := xauthsigning.SignerData{
		ChainID:       Cfg.Networks.Noble.ChainId,
		AccountNumber: accountNumber,
		Sequence:      accountSequence,
	}

	sigV2, err = clientTx.SignWithPrivKey(
		sdkContext.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		priv,
		sdkContext.TxConfig,
		accountSequence,
	)

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return err
	}

	// Generated Protobuf-encoded bytes.
	txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return err
	}

	// broadcast txn

	// set up grpc sdkContext
	grpcConn, err := grpc.Dial(
		Cfg.Networks.Noble.RPC,
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
			Mode:    2,
		},
	)
	if err != nil {
		return err
	}

	if grpcRes.TxResponse.Code != 0 {
		return errors.New(fmt.Sprintf("nonzero error code: %d", grpcRes.TxResponse.Code))
	}

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

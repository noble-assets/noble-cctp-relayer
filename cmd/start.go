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
	"github.com/pascaldekloe/etherstream"
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
	ethClient, err := ethclient.DialContext(context.Background(), Cfg.Networks.Ethereum.RPC)
	if err != nil {
		logger.Error("unable to initialize ethereum ethClient", "err", err)
		os.Exit(1)
	}

	Run(ethClient)

}

// websockets do not query history
// https://github.com/ethereum/go-ethereum/issues/15063
func Run(ethClient *ethclient.Client) {

	// set up clients
	MessageTransmitter = common.HexToAddress(Cfg.Networks.Ethereum.MessageTransmitter)
	etherReader := etherstream.Reader{Backend: ethClient}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{MessageTransmitter},
		Topics:    [][]common.Hash{{MessageSent.ID}},
		FromBlock: big.NewInt(9573850),
		ToBlock:   big.NewInt(9573860),
	}

	stream, sub, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	for _, log := range history {
		attestation := ProcessLog(log)

		if attestation != nil && CheckAttestation(attestation) {
			Mint(attestation)
		}
	}

	fmt.Println()

	for {
		select {
		case err := <-sub.Err():
			logger.Error("connection closed", "err", err)
			os.Exit(1)
		case log := <-stream:
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

	if burn, err := new(types.BurnMessage).Parse(message.MessageBody); err == nil {
		logger.Info("received a new burn message", "nonce", message.Nonce, "tx", log.TxHash)

		hexRaw, _ := hex.DecodeString("000000000000000000000004000000000003950D000000000000000000000000D0C3DA58F55358142B8D3E06C1C30C5C6114EFE800000000000000000000000057D4EAF1091577A6B7D121202AFBD2808134F11700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007865C6E87B9F70255377E024ACE6630C1EAA37F000000000000000000000000580B5AFD4B41B887339EA92D98F88BF07AAF04F500000000000000000000000000000000000000000000000000000000000F4240000000000000000000000000DB86162D6E6B273A95BD4D20FAADB83D7B5FE1CA")
		hashed := crypto.Keccak256(hexRaw)
		hashedHexStr := hex.EncodeToString(hashed) // CORRECT
		fmt.Println(hashedHexStr)

		k1 := message.MessageBody
		k1s := hex.EncodeToString(k1)
		fmt.Println(k1s)
		k2 := crypto.Keccak256(k1)
		k2s := crypto.Keccak256Hash(k2).Hex()
		fmt.Println(k2s)
		k3 := hex.EncodeToString(k2)
		fmt.Println(k3)
		fmt.Println(burn)
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

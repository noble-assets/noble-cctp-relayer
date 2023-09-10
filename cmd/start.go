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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pascaldekloe/etherstream"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"google.golang.org/grpc"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"
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

// iris api lookup id -> MessageState
var state = map[string]types.MessageState{}

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

	// websockets do not query history
	// https://github.com/ethereum/go-ethereum/issues/15063
	stream, sub, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	if err != nil {
		logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	// process history
	for _, log := range history {
		messageState, _ := types.ToMessageState(Cfg, &log)
		go Process(messageState)
	}

	// constantly consume stream updates
	go func() {
		for {
			select {
			case err := <-sub.Err():
				logger.Error("connection closed", "err", err)
				os.Exit(1)
			case log := <-stream:
				messageState, _ := types.ToMessageState(Cfg, &log)
				Process(messageState)
			}
		}
	}()

	// constantly comb through MessageStates
	for {
		time.Sleep(10 * time.Second)
		for _, messageState := range state {
			go Process(&messageState)
		}
	}
}

// Process is the main processing pipeline.  Depending on the
func Process(messageState *types.MessageState) {

	// if we haven't seen this message, add it to the store
	if _, ok := state[messageState.IrisLookupId]; !ok {
		messageState.Status = types.Created
		messageState.Created = time.Now()
		messageState.Updated = time.Now()

		state[messageState.IrisLookupId] = *messageState
	}
	// if the message is burned or pending, check for an attestation
	if messageState.Status == types.Created || messageState.Status == types.Pending {
		response, exists := CheckAttestation(messageState.IrisLookupId)
		if exists {
			if messageState.Status == types.Created && response.Status == "pending" {
				messageState.Status = types.Pending
				messageState.Updated = time.Now()
				return
			} else if response.Status == "complete" {
				messageState.Status = types.Attested
				messageState.Updated = time.Now()
			}
		} else {
			return
		}
	}
	// if the message is attested to, try to mint
	if messageState.Status == types.Attested {
		response, err := Mint(messageState)
		if err != nil {
			logger.Error("unable to mint", "err", err)
			return
		}
		if response.Code != 0 {
			logger.Error("nonzero response code received", "err", err)
			return
		}
		// success!
		messageState.DestTxHash = response.TxHash
		messageState.Status = types.Complete
		messageState.Updated = time.Now()
	}
	// if the message is complete, ignore
	if messageState.Status == types.Complete {
		return
	}
}

// CheckAttestation checks the iris api for attestation status
// returns true if attestation is complete
func CheckAttestation(irisLookupId string) (*types.AttestationResponse, bool) {
	logger.Info(fmt.Sprintf("CheckAttestation for %s%s", Cfg.AttestationBaseUrl, irisLookupId))

	rawResponse, err := http.Get(Cfg.AttestationBaseUrl + irisLookupId)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		logger.Debug("non 200 response received")
		return nil, false
	}
	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		logger.Debug("unable to parse message body")
		return nil, false
	}

	response := types.AttestationResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		logger.Debug("unable to unmarshal response")
		return nil, false
	}

	return &response, true
}

func Mint(messageState *types.MessageState) (*sdktypes.TxResponse, error) {
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

	attestationBz, err := hex.DecodeString(messageState.Attestation)
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}
	msg := nobletypes.NewMsgReceiveMessage(
		"", // TODO
		messageState.MsgSentBytes,
		attestationBz,
	)
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	// Generated Protobuf-encoded bytes.
	txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	// broadcast txn

	// set up grpc sdkContext
	grpcConn, err := grpc.Dial(
		Cfg.Networks.Noble.RPC,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	if grpcRes.TxResponse.Code != 0 {
		return nil, errors.New(fmt.Sprintf("nonzero error code: %d", grpcRes.TxResponse.Code))
	}

	return grpcRes.TxResponse, nil
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

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
	"math/rand"
	"net/http"
	"os"
	"time"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start relaying CCTP transactions from Ethereum to Noble",
	Run:   RelayEthereum,
}

// iris api lookup id -> MessageState
var state = map[string]types.MessageState{}

func RelayEthereum(cmd *cobra.Command, args []string) {

	// set up clients
	ethConfig := Cfg.Networks.Source.Ethereum
	messageTransmitterAddress := common.HexToAddress(ethConfig.MessageTransmitter)
	etherReader := etherstream.Reader{Backend: EthClient}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{messageTransmitterAddress},
		Topics:    [][]common.Hash{{MessageSent.ID}},
		FromBlock: big.NewInt(int64(ethConfig.StartBlock - ethConfig.LookbackPeriod)),
		ToBlock:   big.NewInt(int64(ethConfig.StartBlock)),
	}

	messageSent := MessageTransmitterABI.Events["MessageSent"]

	// websockets do not query history
	// https://github.com/ethereum/go-ethereum/issues/15063
	stream, sub, history, err := etherReader.QueryWithHistory(context.Background(), &query)
	if err != nil {
		Logger.Error("unable to subscribe to logs", "err", err)
		os.Exit(1)
	}

	// process history
	for _, log := range history {
		messageState, err := types.ToMessageState(MessageTransmitterABI, messageSent, &log)
		if err != nil {
			Logger.Error("Unable to parse history log into MessageState, skipping")
			continue
		}
		go Process(messageState)
	}

	// constantly consume stream updates
	go func() {
		for {
			select {
			case err := <-sub.Err():
				Logger.Error("connection closed", "err", err)
				os.Exit(1)
			case log := <-stream:
				messageState, err := types.ToMessageState(MessageTransmitterABI, messageSent, &log)
				if err != nil {
					Logger.Error("Unable to parse ws log into MessageState, skipping")
					continue
				}
				go Process(messageState)
			}
		}
	}()

	// constantly comb through MessageStates
	for {
		time.Sleep(30 * time.Second) // TODO configurable
		for _, messageState := range state {
			go Process(&messageState)
		}
	}
}

// Process is the main processing pipeline.
func Process(messageState *types.MessageState) {

	// if we haven't seen this message, add it to the store
	if _, ok := state[messageState.IrisLookupId]; !ok {
		state[messageState.IrisLookupId] = *messageState
	}

	// filters
	if messageState.FilterDisabledCCTPRoutes(&Cfg) ||
		messageState.FilterInvalidDestinationCallers(&Cfg) {
		messageState.Status = types.Filtered
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
		switch messageState.DestDomain {
		case 0:
			response, err := BroadcastNoble(messageState)
			if err != nil {
				Logger.Error("unable to mint", "err", err)
				return
			}
			if response.Code != 0 {
				Logger.Error("nonzero response code received", "err", err)
				return
			}
			// success!
			messageState.DestTxHash = response.TxHash
		}
		messageState.Status = types.Complete
		messageState.Updated = time.Now()
	}

	// if the message is complete or failed, ignore
	if messageState.Status == types.Complete ||
		messageState.Status == types.Failed ||
		messageState.Status == types.Filtered {
		return
	}
}

// CheckAttestation checks the iris api for attestation status
// returns true if attestation is complete
func CheckAttestation(irisLookupId string) (*types.AttestationResponse, bool) {
	Logger.Info(fmt.Sprintf("CheckAttestation for %s%s", Cfg.AttestationBaseUrl, irisLookupId))

	rawResponse, err := http.Get(Cfg.AttestationBaseUrl + irisLookupId)
	if rawResponse.StatusCode != http.StatusOK || err != nil {
		Logger.Debug("non 200 response received")
		return nil, false
	}
	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		Logger.Debug("unable to parse message body")
		return nil, false
	}

	response := types.AttestationResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		Logger.Debug("unable to unmarshal response")
		return nil, false
	}

	return &response, true
}

// BroadcastNoble broadcasts a message to Noble
func BroadcastNoble(messageState *types.MessageState) (*sdktypes.TxResponse, error) {
	// set up sdk context
	// TODO move this out of BroadcastNoble
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sdkContext := sdkClient.Context{
		ChainID:          Cfg.Networks.Destination.Noble.ChainId,
		TxConfig:         xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
		AccountRetriever: xauthtypes.AccountRetriever{},
		NodeURI:          Cfg.Networks.Destination.Noble.RPC,
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()
	attestationBytes, err := hex.DecodeString(messageState.Attestation)
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}
	msg := nobletypes.NewMsgReceiveMessage(
		"", // TODO
		messageState.MsgSentBytes,
		attestationBytes,
	)
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}

	txBuilder.SetGasLimit(Cfg.Networks.Destination.Noble.GasLimit)
	txBuilder.SetMemo(generateRelayerMessage())

	// sign tx
	priv, _, _ := testdata.KeyTestPubAddr() // TODO delete

	// get account number, sequence
	addrBytes, err := sdktypes.GetFromBech32(Cfg.Networks.Destination.Noble.MinterAddress, "noble")
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
		ChainID:       Cfg.Networks.Destination.Noble.ChainId,
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
		Cfg.Networks.Destination.Noble.RPC,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return nil, err
	}
	defer grpcConn.Close()

	txClient := tx.NewServiceClient(grpcConn)

	Logger.Info(fmt.Sprintf(
		"Broadcasting message for source domain %d to dest domain %d with tx hash %s",
		messageState.SourceDomain,
		messageState.DestDomain,
		messageState.SourceTxHash))

	for attempt := 0; attempt < Cfg.Networks.Destination.Noble.BroadcastRetries; attempt++ {
		grpcRes, err := txClient.BroadcastTx(
			context.Background(),
			&tx.BroadcastTxRequest{
				TxBytes: txBytes,
				Mode:    2,
			},
		)
		if err != nil {
			Logger.Error(fmt.Sprintf("error during broadcasting: %s", err.Error()))
		}
		if grpcRes.TxResponse.Code == 0 {
			return grpcRes.TxResponse, nil
		} else {
			Logger.Info("Failed to broadcast: nonzero error code")
			// retry
			if attempt < Cfg.Networks.Destination.Noble.BroadcastRetryInterval-1 {
				Logger.Info(fmt.Sprintf("Retrying in %d seconds", Cfg.Networks.Destination.Noble.BroadcastRetryInterval))
				time.Sleep(time.Duration(Cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
			}
		}
	}
	return nil, errors.New("reached max number of broadcast attempts")
}

func generateRelayerMessage() string {
	quotes := []string{
		"Your Commie has no regard for human life. Not even his own.",
		"Gee, I wish we had one of them doomsday machines.",
		"Of course, the whole point of a Doomsday machine is lost if you keep it a secret! Why didn't you tell the world?",
		"Well, boys, this is it. Nuclear combat, toe to toe with the Rooskies.",
		"Mister President, we must not allow a mine shaft gap!",
		"Deterrence is the art of producing, in the mind of the enemy...the fear to attack!",
	}
	choice := rand.Intn(len(quotes))
	return quotes[choice]
}

func init() {
	cobra.OnInitialize(func() {
		messageTransmitter, err := os.Open("./abi/MessageTransmitter.json")
		if err != nil {
			Logger.Error("unable to read MessageTransmitter abi", "err", err)
			os.Exit(1)
		}
		MessageTransmitterABI, err = abi.JSON(messageTransmitter)
		if err != nil {
			Logger.Error("unable to parse MessageTransmitter abi", "err", err)
		}

		MessageSent = MessageTransmitterABI.Events["MessageSent"]

		EthClient, err = ethclient.DialContext(context.Background(), Cfg.Networks.Source.Ethereum.RPC)
		if err != nil {
			Logger.Error("unable to initialize ethereum client", "err", err)
			os.Exit(1)
		}
	})
}

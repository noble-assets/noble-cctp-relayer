package noble

import (
	"context"
	"cosmossdk.io/log"
	"encoding/hex"
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
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math/rand"
	"time"
)

// BroadcastNoble broadcasts a message to Noble
func Broadcast(cfg config.Config, logger log.Logger, msg types.MessageState) (*sdktypes.TxResponse, error) {
	// set up sdk context
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sdkContext := sdkClient.Context{
		ChainID:          cfg.Networks.Destination.Noble.ChainId,
		TxConfig:         xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
		AccountRetriever: xauthtypes.AccountRetriever{},
		NodeURI:          cfg.Networks.Destination.Noble.RPC,
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()
	attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
	if err != nil {
		return nil, errors.New("unable to decode message attestation")
	}
	receiveMsg := nobletypes.NewMsgReceiveMessage(
		cfg.Networks.Minters[msg.DestDomain].MinterAddress,
		msg.MsgSentBytes,
		attestationBytes,
	)
	err = txBuilder.SetMsgs(receiveMsg)
	if err != nil {
		return nil, err
	}

	txBuilder.SetGasLimit(cfg.Networks.Destination.Noble.GasLimit)
	txBuilder.SetMemo(generateRelayerMessage())

	// sign tx
	privKey, _, _ := testdata.KeyTestPubAddr()
	//privKey := Cfg.Minters[msg.DestDomain].MinterPrivateKey

	// get account number, sequence
	addrBytes, err := sdktypes.GetFromBech32(cfg.Networks.Minters[msg.DestDomain].MinterAddress, "noble")
	if err != nil {
		return nil, err
	}
	accountNumber, accountSequence, err := sdkContext.AccountRetriever.GetAccountNumberSequence(sdkContext, addrBytes)

	sigV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: accountSequence,
	}

	signerData := xauthsigning.SignerData{
		ChainID:       cfg.Networks.Destination.Noble.ChainId,
		AccountNumber: accountNumber,
		Sequence:      accountSequence,
	}

	sigV2, err = clientTx.SignWithPrivKey(
		sdkContext.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		privKey,
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
		"noble-grpc.polkachu.com:21590",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithDefaultCallOptions(
		//	grpc.ForceCodec(
		//		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()).GRPCCodec())),
	)
	if err != nil {
		return nil, err
	}
	defer grpcConn.Close()

	txClient := tx.NewServiceClient(grpcConn)

	logger.Info(fmt.Sprintf(
		"Broadcasting message for source domain %d to dest domain %d with tx hash %s",
		msg.SourceDomain,
		msg.DestDomain,
		msg.SourceTxHash))

	for attempt := 0; attempt < cfg.Networks.Destination.Noble.BroadcastRetries+1; attempt++ {
		grpcSimRes, err := txClient.Simulate(
			context.Background(),
			&tx.SimulateRequest{TxBytes: txBytes},
		)
		if err != nil {
			fmt.Println(grpcSimRes)
			logger.Error(fmt.Sprintf("error during simulation: %s", err.Error()))
		}

		grpcRes, err := txClient.BroadcastTx(
			context.Background(),
			&tx.BroadcastTxRequest{
				TxBytes: txBytes,
				Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
			},
		)
		if err != nil {
			logger.Error(fmt.Sprintf("error during broadcasting: %s", err.Error()))
		}
		if grpcRes.TxResponse.Code == 0 {
			return grpcRes.TxResponse, nil
		} else {
			logger.Info("Failed to broadcast: nonzero error code")
			// retry
			if attempt < cfg.Networks.Destination.Noble.BroadcastRetryInterval-1 {
				logger.Info(fmt.Sprintf("Retrying in %d seconds", cfg.Networks.Destination.Noble.BroadcastRetryInterval))
				time.Sleep(time.Duration(cfg.Networks.Destination.Noble.BroadcastRetryInterval) * time.Second)
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

package noble

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"cosmossdk.io/log"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/strangelove-ventures/noble-cctp-relayer/relayer"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var (
	regexAccountSequenceMismatchErr = regexp.MustCompile(`expected (\d+), got (\d+)`)
)

func (n *Noble) InitializeBroadcaster(
	ctx context.Context,
	logger log.Logger,
	sequenceMap *types.SequenceMap,
) error {
	accountNumber, accountSequence, err := n.AccountInfo(ctx)
	if err != nil {
		return fmt.Errorf("unable to get account info for noble: %w", err)
	}

	n.accountNumber = accountNumber
	sequenceMap.Put(n.Domain(), accountSequence)

	return nil
}

func (n *Noble) Broadcast(
	ctx context.Context,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
	m *relayer.PromMetrics,
) error {
	// set up sdk context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	nobletypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	sdkContext := sdkclient.Context{
		TxConfig: xauthtx.NewTxConfig(cdc, xauthtx.DefaultSignModes),
	}

	// build txn
	txBuilder := sdkContext.TxConfig.NewTxBuilder()

	// sign and broadcast txn
	for attempt := 1; attempt <= n.maxRetries; attempt++ {
		err := n.attemptBroadcast(ctx, logger, msgs, sequenceMap, sdkContext, txBuilder)
		if err == nil {
			return nil
		}

		// Log retry information
		logger.Error(fmt.Sprintf("Broadcasting to noble failed. Attempt %d/%d Retrying...", attempt, n.maxRetries), "error", err, "interval_seconds", n.retryIntervalSeconds, "src-tx", msgs[0].SourceTxHash)
		time.Sleep(time.Duration(n.retryIntervalSeconds) * time.Second)
	}

	for _, msg := range msgs {
		if msg.Status != types.Complete {
			msg.Status = types.Failed
		}
	}
	if m != nil {
		m.IncBroadcastErrors(n.Name(), fmt.Sprint(n.Domain()))
	}
	return errors.New("reached max number of broadcast attempts")
}

func (n *Noble) attemptBroadcast(
	ctx context.Context,
	logger log.Logger,
	msgs []*types.MessageState,
	sequenceMap *types.SequenceMap,
	sdkContext sdkclient.Context,
	txBuilder sdkclient.TxBuilder,
) error {

	var receiveMsgs []sdk.Msg
	for _, msg := range msgs {

		used, err := n.cc.QueryUsedNonce(ctx, types.Domain(msg.SourceDomain), msg.Nonce)
		if err != nil {
			return fmt.Errorf("unable to query used nonce: %w", err)
		}

		if used {
			msg.Status = types.Complete
			logger.Info(fmt.Sprintf("Noble cctp minter nonce %d already used.", msg.Nonce), "src-tx", msg.SourceTxHash)
			continue
		}

		// check if another worker already broadcasted tx due to flush
		if msg.Status == types.Complete {
			continue
		}

		attestationBytes, err := hex.DecodeString(msg.Attestation[2:])
		if err != nil {
			return fmt.Errorf("unable to decode message attestation")
		}

		receiveMsgs = append(receiveMsgs, nobletypes.NewMsgReceiveMessage(
			n.minterAddress,
			msg.MsgSentBytes,
			attestationBytes,
		))

		logger.Info(fmt.Sprintf(
			"Broadcasting message from %d to %d: with source tx hash %s",
			msg.SourceDomain,
			msg.DestDomain,
			msg.SourceTxHash))
	}

	if len(receiveMsgs) == 0 {
		return nil
	}

	if err := txBuilder.SetMsgs(receiveMsgs...); err != nil {
		return fmt.Errorf("failed to set messages on tx: %w", err)
	}

	txBuilder.SetGasLimit(n.gasLimit)

	txBuilder.SetMemo(n.txMemo)

	n.mu.Lock()
	defer n.mu.Unlock()

	accountSequence := sequenceMap.Next(n.Domain())

	sigV2 := signing.SignatureV2{
		PubKey: n.privateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: uint64(accountSequence),
	}

	signerData := xauthsigning.SignerData{
		ChainID:       n.chainID,
		AccountNumber: uint64(n.accountNumber),
		Sequence:      uint64(accountSequence),
	}

	err := txBuilder.SetSignatures(sigV2)
	if err != nil {
		return fmt.Errorf("failed to set signatures: %w", err)
	}

	sigV2, err = clientTx.SignWithPrivKey(
		sdkContext.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		n.privateKey,
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

	rpcResponse, err := n.cc.RPCClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		return err
	}

	if rpcResponse.Code == 32 {
		newAccountSequence := n.extractAccountSequence(ctx, logger, rpcResponse.Log)
		logger.Debug(fmt.Sprintf("retrying with new account sequence: %d", newAccountSequence))
		sequenceMap.Put(n.Domain(), newAccountSequence)
	}

	if rpcResponse.Code != 0 {
		return fmt.Errorf("received non-zero: %d - %s", rpcResponse.Code, rpcResponse.Log)
	}

	// Tx was successfully broadcast
	for _, msg := range msgs {
		msg.DestTxHash = rpcResponse.Hash.String()
		msg.Status = types.Complete
	}

	logger.Info(fmt.Sprintf("Successfully broadcast %s to Noble.  Tx hash: %s", msgs[0].SourceTxHash, msgs[0].DestTxHash))

	return nil
}

// extractAccountSequence attempts to extract the account sequence number from the RPC response logs when
// account sequence mismatch errors are encountered. If the account sequence number cannot be extracted from the logs,
// it is retrieved by making a request to the API endpoint.
func (n *Noble) extractAccountSequence(ctx context.Context, logger log.Logger, rpcResponseLog string) uint64 {
	match := regexAccountSequenceMismatchErr.FindStringSubmatch(rpcResponseLog)

	if len(match) == 3 {
		// Extract the numbers from the match.
		newAccountSequence, _ := strconv.ParseUint(match[1], 10, 64)
		return newAccountSequence
	}

	// Otherwise, just request the account sequence
	_, newAccountSequence, err := n.AccountInfo(ctx)
	if err != nil {
		logger.Error("unable to retrieve account sequence")
	}

	return newAccountSequence
}

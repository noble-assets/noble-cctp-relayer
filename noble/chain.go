package noble

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/log"
	nobletypes "github.com/circlefin/noble-cctp/x/cctp/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	clientTx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	xauthtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

var _ types.Chain = (*Noble)(nil)

type Noble struct {
	cc      *cosmos.CosmosProvider
	chainID string

	privateKey    *secp256k1.PrivKey
	minterAddress string
	accountNumber uint64

	startBlock     uint64
	lookbackPeriod uint64
	workers        uint32

	gasLimit             uint64
	txMemo               string
	maxRetries           int
	retryIntervalSeconds int

	mu sync.Mutex
}

func NewChain(
	rpcURL string,
	chainID string,
	privateKey string,
	startBlock uint64,
	lookbackPeriod uint64,
	workers uint32,
	gasLimit uint64,
	txMemo string,
	maxRetries int,
	retryIntervalSeconds int,
) (*Noble, error) {
	cc, err := cosmos.NewProvider(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("unable to build cosmos provider for noble: %w", err)
	}

	keyBz, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse noble private key: %w", err)
	}

	privKey := secp256k1.PrivKey{Key: keyBz}

	address := privKey.PubKey().Address()
	minterAddress := sdk.MustBech32ifyAddressBytes("noble", address)

	return &Noble{
		cc:                   cc,
		chainID:              chainID,
		startBlock:           startBlock,
		lookbackPeriod:       lookbackPeriod,
		workers:              workers,
		privateKey:           &privKey,
		minterAddress:        minterAddress,
		gasLimit:             gasLimit,
		txMemo:               txMemo,
		maxRetries:           maxRetries,
		retryIntervalSeconds: retryIntervalSeconds,
	}, nil
}

func (n *Noble) AccountInfo(ctx context.Context) (uint64, uint64, error) {
	res, err := authtypes.NewQueryClient(n.cc).Account(ctx, &authtypes.QueryAccountRequest{
		Address: n.minterAddress,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query account for noble: %w", err)
	}
	var acc authtypes.AccountI
	if err := n.cc.Cdc.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, 0, fmt.Errorf("unable to unpack account for noble: %w", err)
	}

	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

func (n *Noble) Name() string {
	return "Noble"
}

func (n *Noble) Domain() types.Domain {
	return 4
}

func (n *Noble) IsDestinationCaller(destinationCaller []byte) bool {
	zeroByteArr := make([]byte, 32)

	if bytes.Equal(destinationCaller, zeroByteArr) {
		return true
	}

	bech32DestinationCaller, err := decodeDestinationCaller(destinationCaller)
	if err != nil {
		return false
	}

	return bech32DestinationCaller == n.minterAddress
}

// DecodeDestinationCaller transforms an encoded Noble cctp address into a noble bech32 address
// left padded input -> bech32 output
func decodeDestinationCaller(input []byte) (string, error) {
	if len(input) <= 12 {
		return "", errors.New("destinationCaller is too short")
	}
	output, err := bech32.ConvertAndEncode("noble", input[12:])
	if err != nil {
		return "", errors.New("unable to encode destination caller")
	}
	return output, nil
}

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

func (n *Noble) StartListener(
	ctx context.Context,
	logger log.Logger,
	processingQueue chan *types.TxState,
) {
	logger = logger.With("chain", n.Name(), "chain_id", n.chainID, "domain", n.Domain())

	if n.startBlock == 0 {
		// get the latest block
		chainTip, err := n.chainTip(ctx)
		if err != nil {
			panic(fmt.Errorf("unable to get chain tip for noble: %w", err))
		}
		n.startBlock = chainTip
	}

	logger.Info(fmt.Sprintf("Starting Noble listener at block %d looking back %d blocks",
		n.startBlock,
		n.lookbackPeriod))

	accountNumber, _, err := n.AccountInfo(ctx)
	if err != nil {
		panic(fmt.Errorf("unable to get account info for noble: %w", err))
	}

	n.accountNumber = accountNumber

	// enqueue block heights
	currentBlock := n.startBlock
	lookback := n.lookbackPeriod
	chainTip, err := n.chainTip(ctx)
	blockQueue := make(chan uint64, 1000000)

	// history
	currentBlock = currentBlock - lookback
	for currentBlock <= chainTip {
		blockQueue <- currentBlock
		currentBlock++
	}

	// listen for new blocks
	go func() {
		first := make(chan struct{}, 1)
		first <- struct{}{}
		for {
			timer := time.NewTimer(6 * time.Second)
			select {
			case <-first:
				timer.Stop()
				chainTip, err = n.chainTip(ctx)
				if err == nil {
					if chainTip >= currentBlock {
						for i := currentBlock; i <= chainTip; i++ {
							blockQueue <- i
						}
						currentBlock = chainTip + 1
					}
				}
			case <-timer.C:
				chainTip, err = n.chainTip(ctx)
				if err == nil {
					if chainTip >= currentBlock {
						for i := currentBlock; i <= chainTip; i++ {
							blockQueue <- i
						}
						currentBlock = chainTip + 1
					}
				}
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()

	// constantly query for blocks
	for i := 0; i < int(n.workers); i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					block := <-blockQueue
					res, err := n.cc.RPCClient.TxSearch(ctx, fmt.Sprintf("tx.height=%d", block), false, nil, nil, "")
					if err != nil {
						logger.Debug(fmt.Sprintf("unable to query Noble block %d", block))
						blockQueue <- block
					}

					for _, tx := range res.Txs {
						parsedMsgs, err := txToMessageState(tx)
						if err != nil {
							logger.Error("unable to parse Noble log to message state", "err", err.Error())
							continue
						}
						for _, parsedMsg := range parsedMsgs {
							logger.Info(fmt.Sprintf("New stream msg with nonce %d from %d with tx hash %s", parsedMsg.Nonce, parsedMsg.SourceDomain, parsedMsg.SourceTxHash))
						}
						processingQueue <- &types.TxState{TxHash: tx.Hash.String(), Msgs: parsedMsgs}
					}
				}
			}
		}()
	}

	<-ctx.Done()
}

func (n *Noble) chainTip(ctx context.Context) (uint64, error) {
	res, err := n.cc.RPCClient.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to query status for noble: %w", err)
	}
	return uint64(res.SyncInfo.LatestBlockHeight), nil
}

func (n *Noble) Broadcast(
	ctx context.Context,
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

	// sign and broadcast txn
	for attempt := 0; attempt <= n.maxRetries; attempt++ {

		//TODO: MOVE EVERYTHING IN FOR LOOP TO FUNCTION. Same for ETH.
		// see todo below.

		var receiveMsgs []sdk.Msg
		for _, msg := range msgs {

			used, err := n.cc.QueryUsedNonce(ctx, types.Domain(msg.SourceDomain), msg.Nonce)
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

		if err := txBuilder.SetMsgs(receiveMsgs...); err != nil {
			return fmt.Errorf("failed to set messages on tx: %w", err)
		}

		txBuilder.SetGasLimit(n.gasLimit)

		txBuilder.SetMemo(n.txMemo)

		n.mu.Lock()
		// TODO: uncomment this & remove all remainin n.mu.Unlock() 's after moving loop body to its own function
		// defer n.mu.Unlock()

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

		txBuilder.SetSignatures(sigV2)

		sigV2, err := clientTx.SignWithPrivKey(
			sdkContext.TxConfig.SignModeHandler().DefaultMode(),
			signerData,
			txBuilder,
			n.privateKey,
			sdkContext.TxConfig,
			uint64(accountSequence),
		)
		if err != nil {
			n.mu.Unlock()
			return fmt.Errorf("failed to sign tx: %w", err)
		}

		if err := txBuilder.SetSignatures(sigV2); err != nil {
			n.mu.Unlock()
			return fmt.Errorf("failed to set signatures: %w", err)
		}

		// Generated Protobuf-encoded bytes.
		txBytes, err := sdkContext.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			n.mu.Unlock()
			return fmt.Errorf("failed to proto encode tx: %w", err)
		}

		rpcResponse, err := n.cc.RPCClient.BroadcastTxSync(ctx, txBytes)
		if err != nil || (rpcResponse != nil && rpcResponse.Code != 0) {
			// Log the error
			logger.Error(fmt.Sprintf("error during broadcast: %s", getErrorString(err, rpcResponse)))

			if err != nil || rpcResponse == nil {
				// Log retry information
				logger.Info(fmt.Sprintf("Retrying in %d seconds", n.retryIntervalSeconds))
				time.Sleep(time.Duration(n.retryIntervalSeconds) * time.Second)
				// wait a random amount of time to lower probability of concurrent message nonce collision
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
				n.mu.Unlock()
				continue
			}

			// Log details for non-zero response code
			logger.Error(fmt.Sprintf("received non-zero: %d - %s", rpcResponse.Code, rpcResponse.Log))

			// Handle specific error code (32)
			if rpcResponse.Code == 32 {
				newAccountSequence := n.extractAccountSequence(ctx, logger, rpcResponse.Log)
				logger.Debug(fmt.Sprintf("retrying with new account sequence: %d", newAccountSequence))
				sequenceMap.Put(n.Domain(), newAccountSequence)
			}

			// Log retry information
			logger.Info(fmt.Sprintf("Retrying in %d seconds", n.retryIntervalSeconds))
			time.Sleep(time.Duration(n.retryIntervalSeconds) * time.Second)
			// wait a random amount of time to lower probability of concurrent message nonce collision
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			n.mu.Unlock()
			continue
		}

		n.mu.Unlock()

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
func (n *Noble) extractAccountSequence(ctx context.Context, logger log.Logger, rpcResponseLog string) uint64 {
	pattern := `expected (\d+), got (\d+)`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(rpcResponseLog)

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

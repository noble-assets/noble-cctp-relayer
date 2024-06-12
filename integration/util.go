package integration_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/strangelove-ventures/noble-cctp-relayer/cosmos"
)

//nolint:gosec
const (
	usdcTokenAddressSepolia      = "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"
	TokenMessengerAddressSepolia = "0x9f3B8679c73C2Fef8b59B4f3444d4e156fb70AA5"
	uusdcDenom                   = "uusdc"
)

// nolint:unparam
func getNobleAccountBalance(ctx context.Context, cc *cosmos.CosmosProvider, address, denom string) (uint64, error) {
	qc := bankTypes.NewQueryClient(cc)
	res, err := qc.Balance(ctx, &bankTypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return 0, err
	}

	return res.Balance.Amount.Uint64(), nil
}

func getNobleAccountNumberSequenceGRPC(cc *cosmos.CosmosProvider, address string) (uint64, uint64, error) {
	res, err := authtypes.NewQueryClient(cc).Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query account for noble: %w", err)
	}
	var acc authtypes.AccountI
	if err := cc.Cdc.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, 0, fmt.Errorf("unable to unpack account for noble: %w", err)
	}

	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

func getEthBalance(client *ethclient.Client, usdcTokenAddress, walletAddress string) (uint64, error) {
	accountAddress := common.HexToAddress(walletAddress)
	tokenAddress := common.HexToAddress(usdcTokenAddress)
	erc20ABI := `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return 0, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	data, err := parsedABI.Pack("balanceOf", accountAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to pack data into ABI interface: %v", err)
	}

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{To: &tokenAddress, Data: data}, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %v", err)
	}

	balance := new(big.Int)
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		log.Fatalf("Failed to unpack data from ABI interface: %v", err)
	}

	// Convert to uint64
	return balance.Uint64(), nil
}

// ethConvertPrivateKeytoAddress gets the public address from a hex encoded private key
func ethConvertPrivateKeytoAddress(privateKeyHex string) (string, error) {
	// Decode hex-encoded private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", err
	}

	// Parse the private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", err
	}

	// Get the public key from the private key
	publicKey := privateKey.Public()

	// Convert the public key to the ECDSA format
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("Error converting public key to ECDSA format")
	}

	// Generate Ethereum address
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return address, nil
}

func generateEthWallet() (address, privateKeyHex string, err error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", err
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)

	privateKeyHex = hexutil.Encode(privateKeyBytes)[2:]

	publicKey := privateKey.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", fmt.Errorf("error casting public key to ECDSA")
	}

	address = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return address, privateKeyHex, nil
}

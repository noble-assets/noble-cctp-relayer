package ethereum

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

type JsonError interface {
	Error() string
	ErrorCode() int
	ErrorData() interface{}
}

func GetEthereumAccountNonce(endpoint string, address string) (int64, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to Ethereum RPC at %s: %v", endpoint, err)
	}

	var result string
	if err := client.Call(&result, "eth_getTransactionCount", address, "pending"); err != nil {
		return 0, fmt.Errorf("failed to get transaction count: %v", err)
	}

	nonce := new(big.Int)
	nonce.SetString(result[2:], 16) // Removing "0x" prefix and converting hex string to *big.Int

	return nonce.Int64(), nil
}

// GetEcdsaKeyAddress returns the public ecdsa key and address given the private key
func GetEcdsaKeyAddress(privateKey string) (*ecdsa.PrivateKey, string, error) {
	privEcdsaKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, "", errors.New("unable to convert private key hex to ecdsa")
	}

	publicKey := privEcdsaKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, "", errors.New("error casting public key to ECDSA")
	}

	return privEcdsaKey, crypto.PubkeyToAddress(*publicKeyECDSA).Hex(), nil
}

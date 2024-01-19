package ethereum

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ContractBackendWrapper struct {
	*ethclient.Client
}

func NewContractBackendWrapper(client *ethclient.Client) *ContractBackendWrapper {
	return &ContractBackendWrapper{
		Client: client,
	}
}

func (c *ContractBackendWrapper) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	json, err := tx.MarshalJSON()
	if err != nil {
		return err
	}
	fmt.Printf("SendTransaction: %+v\n\nRAW: %s\n", tx, json)
	return c.Client.SendTransaction(ctx, tx)
}

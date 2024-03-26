package ethereum

import (
	"context"

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
	return c.Client.SendTransaction(ctx, tx)
}

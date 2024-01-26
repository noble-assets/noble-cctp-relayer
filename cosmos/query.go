package cosmos

import (
	"context"
	"fmt"

	cctptypes "github.com/circlefin/noble-cctp/x/cctp/types"
	abci "github.com/cometbft/cometbft/abci/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

// func defaultPageRequest() *querytypes.PageRequest {
// 	return &querytypes.PageRequest{
// 		Key:        []byte(""),
// 		Offset:     0,
// 		Limit:      1000,
// 		CountTotal: false,
// 	}
// }

// QueryABCI performs an ABCI query and returns the appropriate response and error sdk error code.
func (cc *CosmosProvider) QueryABCI(ctx context.Context, req abci.RequestQuery) (abci.ResponseQuery, error) {
	opts := rpcclient.ABCIQueryOptions{
		Height: req.Height,
		Prove:  req.Prove,
	}
	result, err := cc.RPCClient.ABCIQueryWithOptions(ctx, req.Path, req.Data, opts)
	if err != nil {
		return abci.ResponseQuery{}, err
	}

	if !result.Response.IsOK() {
		return abci.ResponseQuery{}, sdkErrorToGRPCError(result.Response)
	}

	return result.Response, nil
}

func (cc *CosmosProvider) QueryUsedNonce(ctx context.Context, sourceDomain types.Domain, nonce uint64) (bool, error) {
	qc := cctptypes.NewQueryClient(cc)

	params := &cctptypes.QueryGetUsedNonceRequest{
		SourceDomain: uint32(sourceDomain),
		Nonce:        nonce,
	}

	_, err := qc.UsedNonce(ctx, params)
	if err != nil {
		if err.Error() == "rpc error: code = NotFound desc = rpc error: code = NotFound desc = not found: key not found" {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// QueryLatestHeight queries the latest height from the RPC client
func (cc *CosmosProvider) QueryLatestHeight(ctx context.Context) (int64, error) {
	status, err := cc.RPCClient.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// GetBlockAtHeight queries the block at a given height
func (cc *CosmosProvider) GetBlockAtHeight(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	block, err := cc.RPCClient.Block(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("error querying block at height %d: %w", height, err)
	}
	return block, nil
}

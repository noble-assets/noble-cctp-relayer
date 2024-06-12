package cosmos

import (
	"time"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	libclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
)

var _ gogogrpc.ClientConn = &CosmosProvider{}

var protoCodec = encoding.GetCodec(proto.Name)

type CosmosProvider struct {
	Cdc       Codec
	RPCClient rpcclient.Client
}

// NewProvider validates the CosmosProviderConfig, instantiates a ChainClient and then instantiates a CosmosProvider
func NewProvider(rpcURL string) (*CosmosProvider, error) {
	rpcClient, err := newRPCClient(rpcURL, 5*time.Second)
	if err != nil {
		return nil, err
	}

	cp := &CosmosProvider{
		Cdc:       makeCodec(ModuleBasics),
		RPCClient: rpcClient,
	}

	return cp, nil
}

// NewRPCClient initializes a new tendermint RPC client connected to the specified address.
func newRPCClient(addr string, timeout time.Duration) (*rpchttp.HTTP, error) {
	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = timeout
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}
	return rpcClient, nil
}

package noble_test

import (
	"cosmossdk.io/log"
	"encoding/hex"
	"github.com/rs/zerolog"
	"github.com/strangelove-ventures/noble-cctp-relayer/cmd/noble"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var cfg config.Config
var logger log.Logger

func init() {
	cfg.AttestationBaseUrl = "https://iris-api-sandbox.circle.com/attestations/"
	cfg.Networks.Destination.Noble.ChainId = "grand-1"
	cfg.Networks.Destination.Noble.RPC = "rpc.testnet.noble.strange.love:26657"
	cfg.Networks.Destination.Noble.BroadcastRetries = 1

	logger = log.NewLogger(os.Stdout, log.LevelOption(zerolog.ErrorLevel))
	cfg.Minters = map[uint32]struct {
		MinterAddress    string "yaml:\"minter-address\""
		MinterPrivateKey string "yaml:\"minter-private-key\""
	}{}

	cfg.Minters[4] = struct {
		MinterAddress    string "yaml:\"minter-address\""
		MinterPrivateKey string "yaml:\"minter-private-key\""
	}{
		MinterAddress:    "noble1wa5g4at8yfmph96jxsvn0ynnf5qx73h0l6ecrs",
		MinterPrivateKey: "",
	}

}

func TestBroadcastNobleSuccess(t *testing.T) {
	msgBz, err := hex.DecodeString("000000000000000000000004000000000003962E000000000000000000000000D0C3DA58F55358142B8D3E06C1C30C5C6114EFE800000000000000000000000057D4EAF1091577A6B7D121202AFBD2808134F11700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007865C6E87B9F70255377E024ACE6630C1EAA37F00000000000000000000000077688AF56722761B975234193792734D006F46EF0000000000000000000000000000000000000000000000000000000000004E20000000000000000000000000971C54A6EB782FACCD00BC3ED5E934CC5BD8E3EF")
	require.Nil(t, err)
	msg := types.MessageState{
		Attestation:  "0x05de330f3eb03ac2a9a1acffca11ac9462cccd442b65a959c7c7afc7ca987fb125d0fdf3095d8b9073f5aeb892bd6cf9af81925b65df9dd783158aaba854e2881bf0b3b0250a63ef2de6106f69f8bbd7c30dfccf6a571a6edd053efa5019aec42c54fb2bbba5a862c54d6061ea45c6886fd98226986e44e2872d9ceaaf8cf5dfe81c",
		DestDomain:   4,
		MsgSentBytes: msgBz,
	}

	txResponse, err := noble.Broadcast(cfg, logger, msg)
	require.Nil(t, err)
	require.Equal(t, txResponse, 0)

}

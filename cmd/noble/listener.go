package noble

import (
	"cosmossdk.io/log"
	"fmt"
	"github.com/strangelove-ventures/noble-cctp-relayer/config"
	"github.com/strangelove-ventures/noble-cctp-relayer/types"
)

func StartListener(cfg config.Config, logger log.Logger, processingQueue chan *types.MessageState) {
	// set up client

	logger.Info(fmt.Sprintf(
		"Starting Noble listener at block %d looking back %d blocks",
		cfg.Networks.Source.Noble.StartBlock,
		cfg.Networks.Source.Noble.LookbackPeriod))

	// constantly query for blocks

}

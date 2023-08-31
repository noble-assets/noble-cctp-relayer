package utils

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func InjectMessages(client *ethclient.Client, filter ethereum.FilterQuery, channel chan types.Log) {
	var hash common.Hash
	var logs []types.Log

	hash = common.HexToHash("0x25bb655b3303a7da24b63aaedf515dca93feb83da52517c17176c03aee6fd545")
	filter.BlockHash = &hash
	logs, _ = client.FilterLogs(context.Background(), filter)

	for _, log := range logs {
		channel <- log
	}

	hash = common.HexToHash("0x26afbc3e913e8779e6f164bd53f4419ae3e893aeff18ffa37e5d4a656b2e7730")
	filter.BlockHash = &hash
	logs, _ = client.FilterLogs(context.Background(), filter)

	for _, log := range logs {
		channel <- log
	}
}

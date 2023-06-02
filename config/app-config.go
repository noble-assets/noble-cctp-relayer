package config

type Config struct {
	Networks           networks
	Indexer            indexer
	ConfigFileLocation string
}

type networks struct {
	Ethereum ethereum
	Noble    noble
}

type ethereum struct {
	RPC                string
	TokenMessenger     string
	MessageTransmitter string
}

type noble struct {
	RPC           string
	DestinationId uint32
}

type indexer struct {
	StartBlock          int64
	AttestationBaseUrl  string
	ValidTokenAddresses map[string]bool
}

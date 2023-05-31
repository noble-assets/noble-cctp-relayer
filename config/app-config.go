package config

type Config struct {
	Networks           networks
	ConfigFileLocation string
}

type networks struct {
	Ethereum network
	Noble    network
}

type network struct {
	RPC string
}

# CCTP Relayer

<p align="center"><img src=".github/assets/portal.png"></p>

This service listens and forwards Cross Chain Transfer Protocol events.   
Lightweight and easily extensible with more chains.

Installation
```shell
git clone https://github.com/strangelove-ventures/noble-cctp-relayer
cd noble-cctp-relayer
make install
```

Running the relayer
```shell
noble-cctp-relayer start --config ./config/sample-app-config.yaml
```
Sample configs can be found in [config](config).

### Flush Interval

Using the `--flush-interval` flag will run a flush on all paths every `duration`; ex `--flush-interval 5m`

The relayer will keep track of the latest flushed block. The first time the flush is run, the flush will start at the chains latest height - lookback period and flush up until height of the chain when the flush started. It will then store the height the flush ended on.

After that, it will flush from the last stored height - lookback period up until the latest height of the chain.

### Prometheus Metrics

By default, metrics are exported at on port :2112/metrics (`http://localhost:2112/metrics`). You can customize the port using the `--metrics-port` flag. 

| **Exported Metric**                 | **Description**                                                                                                                                    | **Type** |
|-------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| cctp_relayer_wallet_balance         | Current balance of a relayer wallet in Wei.<br><br>Noble balances are not currently exported b/c `MsgReceiveMessage` is free to submit on Noble.   | Gauge    |
| cctp_relayer_chain_latest_height    | Current height of the chain.                                                                                                                       | Gauge    |
| cctp_relayer_broadcast_errors_total | The total number of failed broadcasts. Note: this is AFTER it retries `broadcast-retries` (config setting) number of times.                        | Counter  |

### Noble Key

The noble private key you input into the config must be hex encoded. The easiest way to get this is via a chain binary:

`nobled keys export <KEY_NAME> --unarmored-hex --unsafe`


### API
Simple API to query message state cache
```shell
# All messages for a source tx hash
localhost:8000/tx/<hash, including the 0x prefix>
# All messages for a tx hash and domain 0 (Ethereum)
localhost:8000/tx/<hash>?domain=0
```

### State

| IrisLookupId | Status   | SourceDomain | DestDomain | SourceTxHash  | DestTxHash | MsgSentBytes | Created | Updated |
|:-------------|:---------|:-------------|:-----------|:--------------|:-----------|:-------------|:--------|:--------|
| 0x123        | Created  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Pending  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Attested | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Complete | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Failed   | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Filtered | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |

### Generating Go ABI bindings

```shell
abigen --abi ethereum/abi/TokenMessenger.json --pkg contracts --type TokenMessenger --out ethereum/contracts/TokenMessenger.go
abigen --abi ethereum/abi/TokenMessengerWithMetadata.json --pkg contracts --type TokenMessengerWithMetadata --out ethereum/contracts/TokenMessengerWithMetadata.go
abigen --abi ethereum/abi/ERC20.json --pkg integration_testing --type ERC20 --out integration/ERC20.go
abigen --abi ethereum/abi/MessageTransmitter.json --pkg contracts- --type MessageTransmitter --out ethereum/contracts/MessageTransmitter.go
```

### Useful links
[Relayer Flow Charts](./docs/flows.md)

[USDC faucet](https://usdcfaucet.com/)

[Circle Docs/Contract Addresses](https://developers.circle.com/stablecoins/docs/evm-smart-contracts)

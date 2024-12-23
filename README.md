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

Using the `--flush-interval` flag will run a flush on all chains every `duration`; ex `--flush-interval 5m`

The first time the flush is run per chain, the flush will start at the chains `latest height - (2 * lookback period)`. The flush will always finish at the `latest chain height - lookback period`. This allows the flush to lag behind the chain so that the flush does not compete for transactions that are actively being processed. For subsequent flushes, each chain will reference its last flushed block, start from there and flush to the `latest chain height - lookback period` again. The flushing process will continue as long as the relayer is running.

For best results and coverage, the lookback period in blocks should correspond to the flush interval. If a chain produces 1 block a second and the flush interval is set to 30 minutes (1800 seconds), the lookback period should be at least 1800 blocks. When in doubt, round up and add a small buffer.

#### Examples

Consider a 30 minute flush interval (1800 seconds)
- Ethereum: 12 second blocks = (1800 / 12) = `150 blocks`
- Polygon: 2 second blocks = (1800 / 2) = `900 blocks`
- Arbitrum: 0.26 second blocks = (1800 / 0.26) = `~6950 blocks`

### Flush Only Mode

This relayer also supports a `--flush-only-mode`. This mode will only flush the chain and not actively listen for new events as they occur. This is useful for running a secondary relayer which "lags" behind the primary relayer. It is only responsible for retrying failed transactions. 

When the relayer is in flush only mode, the flush mechanism will start at `latest height - (4 * lookback period)` and finish at `latest height - (3 * lookback period)`. For all subsequent flushes, the relayer will start at the last flushed block and finish at `latest height - (3 * lookback period)`. Please see the notes above for configuring the flush interval and lookback period.

> Note: It is highly recommended to use the same configuration for both the primary and secondary relayer. This ensures that there is zero overlap between the relayers.

### Prometheus Metrics

By default, metrics are exported at on port :2112/metrics (`http://localhost:2112/metrics`). You can customize the port using the `--metrics-port` flag. 

| **Exported Metric**                 | **Description**                                                                                                                                  | **Type** |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| cctp_relayer_wallet_balance         | Current balance of a relayer wallet in Wei.<br><br>Noble balances are not currently exported b/c `MsgReceiveMessage` is free to submit on Noble. | Gauge    |
| cctp_relayer_chain_latest_height    | Current height of the chain.                                                                                                                     | Gauge    |
| cctp_relayer_broadcast_errors_total | The total number of failed broadcasts. Note: this is AFTER it retries `broadcast-retries` (config setting) number of times.                      | Counter  |

### Minter Private Keys
Minter private keys are required on a per chain basis to broadcast transactions to the target chain. These private keys can either be set in the `config.yaml` or via environment variables. 

#### Config Private Keys

Please see `./config/sample-config.yaml` for setting minter private keys in configuration. Please note that this method is insecure as the private keys are stored in plain text.

#### Env Vars Private Keys

To pass in a private key via an environment variable, first identify the chain's name. A chain's name corresponds to the key under the `chains` section in the `config.yaml`. The sample config lists these chain names for example: `noble`, `ethereum`, `optimism`, etc. Now, take the chain name in all caps and append `_PRIV_KEY`.

An environment variable for `noble` would look like: `NOBLE_PRIV_KEY=<PRIVATE_KEY_HERE>`

#### Noble Private Key Format

The noble private key you input into the config or via enviroment variables must be hex encoded. The easiest way to get this is via a chain binary:

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

| IrisLookupId | Status   | SourceDomain | DestDomain | SourceTxHash | DestTxHash | MsgSentBytes | Created | Updated |
| :----------- | :------- | :----------- | :--------- | :----------- | :--------- | :----------- | :------ | :------ |
| 0x123        | Created  | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |
| 0x123        | Pending  | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |
| 0x123        | Attested | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |
| 0x123        | Complete | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |
| 0x123        | Failed   | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |
| 0x123        | Filtered | 0            | 4          | 0x123        | ABC123     | bytes...     | date    | date    |

### Generating Go ABI bindings

```shell
abigen --abi ethereum/abi/TokenMessenger.json --pkg contracts --type TokenMessenger --out ethereum/contracts/TokenMessenger.go
abigen --abi ethereum/abi/TokenMessengerWithMetadata.json --pkg contracts --type TokenMessengerWithMetadata --out ethereum/contracts/TokenMessengerWithMetadata.go
abigen --abi ethereum/abi/ERC20.json --pkg integration_testing --type ERC20 --out integration/ERC20.go
abigen --abi ethereum/abi/MessageTransmitter.json --pkg contracts- --type MessageTransmitter --out ethereum/contracts/MessageTransmitter.go
```

### Useful links
[Relayer Flow Charts](./docs/flows.md)

[USDC faucet](https://faucet.circle.com)

[Circle Docs/Contract Addresses](https://developers.circle.com/stablecoins/docs/evm-smart-contracts)

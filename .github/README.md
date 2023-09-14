# CCTP Relayer

<p align="center"><img src="assets/portal.png"></p>

CCTP Relayer is a simple service which listens for events on Ethereum and forwards them to Noble.  
It is meant to be used in conjunction with Circle's Cross Chain Transfer Protocol.

Installation
```shell
...
```

Running the relayer
```shell
rly start --config testnet.yaml
```

# Generating ABI Go bindings

```shell
./abi/abigen --abi ./abi/TokenMessenger.json --pkg cmd --type TokenMessenger --out TokenMessenger.go
./abi/abigen --abi ./abi/ERC20.json --pkg integration_testing --type ERC20 --out ERC20.go
```


Store

| IrisLookupId | Type    | Status   | SourceDomain | DestDomain | SourceTxHash  | DestTxHash | MsgSentBytes | Created | Updated |
|:-------------|:--------|:---------|:-------------|:-----------|:--------------|:-----------|:-------------|:--------|:--------|
| 0x123        | Mint    | Created  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Pending  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Attested | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Complete | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Failed   | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |


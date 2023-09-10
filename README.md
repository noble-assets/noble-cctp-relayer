# CCTP Relayer

<img src="assets/header.png">

CCTP Relayer is a simple service which listens for events on Ethereum and forwards them to Noble.  It is meant to be used in conjunction Circle's Cross Chain Transfer Protocol.

Installation
```shell
...
```

Running the relayer
```shell
rly start --config testnet.yaml
```

Architecture

<img src="assets/indexer.jpg">

Store

| IrisLookupId | Type    | Status   | SourceDomain | DestDomain | SourceTxHash  | DestTxHash | MsgSentBytes | Created | Updated |
|:-------------|:--------|:---------|:-------------|:-----------|:--------------|:-----------|:-------------|:--------|:--------|
| 0x123        | Mint    | Burned   | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Pending  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Attested | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Complete | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |


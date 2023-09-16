# CCTP Relayer

<p align="center"><img src=".github/assets/portal.png"></p>

This service listens for Cross Chain Transfer Protocol events and forwards them to their destinations.   
It is lightweight and extensible.  Other source/destination chains can be easily added.

Installation
```shell
git clone https://github.com/strangelove-ventures/noble-cctp-relayer
cd noble-cctp-relayer
go install
```

Running the relayer
```shell
noble-cctp-relayer start --config testnet.yaml
```

# Generating ABI Go bindings

```shell
abi/abigen --abi abi/TokenMessenger.json --pkg cmd --type TokenMessenger --out cmd/TokenMessenger.go
abi/abigen --abi abi/ERC20.json --pkg integration_testing --type ERC20 --out integration/ERC20.go
```


Store

| IrisLookupId | Type    | Status   | SourceDomain | DestDomain | SourceTxHash  | DestTxHash | MsgSentBytes | Created | Updated |
|:-------------|:--------|:---------|:-------------|:-----------|:--------------|:-----------|:-------------|:--------|:--------|
| 0x123        | Mint    | Created  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Pending  | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Attested | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Forward | Complete | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Failed   | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |
| 0x123        | Mint    | Filtered | 0            | 4          | 0x123         | ABC123     | bytes...     | date    | date    |


// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TokenMessengerWithMetadataMetaData contains all meta data concerning the TokenMessengerWithMetadata contract.
var TokenMessengerWithMetadataMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_tokenMessenger\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_domainNumber\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"_domainRecipient\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"uint64\",\"name\":\"metadataNonce\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"}],\"name\":\"DepositForBurnMetadata\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"channel\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"destinationBech32Prefix\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"destinationRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"mintRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"burnToken\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"memo\",\"type\":\"bytes\"}],\"name\":\"depositForBurn\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"channel\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"destinationBech32Prefix\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"destinationRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"mintRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"burnToken\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"destinationCaller\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"memo\",\"type\":\"bytes\"}],\"name\":\"depositForBurnWithCaller\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domainNumber\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"domainRecipient\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messageTransmitter\",\"outputs\":[{\"internalType\":\"contractMessageTransmitter\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"mintRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"burnToken\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"}],\"name\":\"rawDepositForBurn\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"mintRecipient\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"burnToken\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"destinationCaller\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"}],\"name\":\"rawDepositForBurnWithCaller\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"tokenMessenger\",\"outputs\":[{\"internalType\":\"contractTokenMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// TokenMessengerWithMetadataABI is the input ABI used to generate the binding from.
// Deprecated: Use TokenMessengerWithMetadataMetaData.ABI instead.
var TokenMessengerWithMetadataABI = TokenMessengerWithMetadataMetaData.ABI

// TokenMessengerWithMetadata is an auto generated Go binding around an Ethereum contract.
type TokenMessengerWithMetadata struct {
	TokenMessengerWithMetadataCaller     // Read-only binding to the contract
	TokenMessengerWithMetadataTransactor // Write-only binding to the contract
	TokenMessengerWithMetadataFilterer   // Log filterer for contract events
}

// TokenMessengerWithMetadataCaller is an auto generated read-only Go binding around an Ethereum contract.
type TokenMessengerWithMetadataCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenMessengerWithMetadataTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TokenMessengerWithMetadataTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenMessengerWithMetadataFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TokenMessengerWithMetadataFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TokenMessengerWithMetadataSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TokenMessengerWithMetadataSession struct {
	Contract     *TokenMessengerWithMetadata // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// TokenMessengerWithMetadataCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TokenMessengerWithMetadataCallerSession struct {
	Contract *TokenMessengerWithMetadataCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// TokenMessengerWithMetadataTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TokenMessengerWithMetadataTransactorSession struct {
	Contract     *TokenMessengerWithMetadataTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// TokenMessengerWithMetadataRaw is an auto generated low-level Go binding around an Ethereum contract.
type TokenMessengerWithMetadataRaw struct {
	Contract *TokenMessengerWithMetadata // Generic contract binding to access the raw methods on
}

// TokenMessengerWithMetadataCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TokenMessengerWithMetadataCallerRaw struct {
	Contract *TokenMessengerWithMetadataCaller // Generic read-only contract binding to access the raw methods on
}

// TokenMessengerWithMetadataTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TokenMessengerWithMetadataTransactorRaw struct {
	Contract *TokenMessengerWithMetadataTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTokenMessengerWithMetadata creates a new instance of TokenMessengerWithMetadata, bound to a specific deployed contract.
func NewTokenMessengerWithMetadata(address common.Address, backend bind.ContractBackend) (*TokenMessengerWithMetadata, error) {
	contract, err := bindTokenMessengerWithMetadata(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TokenMessengerWithMetadata{TokenMessengerWithMetadataCaller: TokenMessengerWithMetadataCaller{contract: contract}, TokenMessengerWithMetadataTransactor: TokenMessengerWithMetadataTransactor{contract: contract}, TokenMessengerWithMetadataFilterer: TokenMessengerWithMetadataFilterer{contract: contract}}, nil
}

// NewTokenMessengerWithMetadataCaller creates a new read-only instance of TokenMessengerWithMetadata, bound to a specific deployed contract.
func NewTokenMessengerWithMetadataCaller(address common.Address, caller bind.ContractCaller) (*TokenMessengerWithMetadataCaller, error) {
	contract, err := bindTokenMessengerWithMetadata(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TokenMessengerWithMetadataCaller{contract: contract}, nil
}

// NewTokenMessengerWithMetadataTransactor creates a new write-only instance of TokenMessengerWithMetadata, bound to a specific deployed contract.
func NewTokenMessengerWithMetadataTransactor(address common.Address, transactor bind.ContractTransactor) (*TokenMessengerWithMetadataTransactor, error) {
	contract, err := bindTokenMessengerWithMetadata(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TokenMessengerWithMetadataTransactor{contract: contract}, nil
}

// NewTokenMessengerWithMetadataFilterer creates a new log filterer instance of TokenMessengerWithMetadata, bound to a specific deployed contract.
func NewTokenMessengerWithMetadataFilterer(address common.Address, filterer bind.ContractFilterer) (*TokenMessengerWithMetadataFilterer, error) {
	contract, err := bindTokenMessengerWithMetadata(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TokenMessengerWithMetadataFilterer{contract: contract}, nil
}

// bindTokenMessengerWithMetadata binds a generic wrapper to an already deployed contract.
func bindTokenMessengerWithMetadata(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TokenMessengerWithMetadataMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenMessengerWithMetadata.Contract.TokenMessengerWithMetadataCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.TokenMessengerWithMetadataTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.TokenMessengerWithMetadataTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TokenMessengerWithMetadata.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.contract.Transact(opts, method, params...)
}

// DomainNumber is a free data retrieval call binding the contract method 0x52d440ea.
//
// Solidity: function domainNumber() view returns(uint32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCaller) DomainNumber(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _TokenMessengerWithMetadata.contract.Call(opts, &out, "domainNumber")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// DomainNumber is a free data retrieval call binding the contract method 0x52d440ea.
//
// Solidity: function domainNumber() view returns(uint32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) DomainNumber() (uint32, error) {
	return _TokenMessengerWithMetadata.Contract.DomainNumber(&_TokenMessengerWithMetadata.CallOpts)
}

// DomainNumber is a free data retrieval call binding the contract method 0x52d440ea.
//
// Solidity: function domainNumber() view returns(uint32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCallerSession) DomainNumber() (uint32, error) {
	return _TokenMessengerWithMetadata.Contract.DomainNumber(&_TokenMessengerWithMetadata.CallOpts)
}

// DomainRecipient is a free data retrieval call binding the contract method 0xe95f1219.
//
// Solidity: function domainRecipient() view returns(bytes32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCaller) DomainRecipient(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _TokenMessengerWithMetadata.contract.Call(opts, &out, "domainRecipient")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DomainRecipient is a free data retrieval call binding the contract method 0xe95f1219.
//
// Solidity: function domainRecipient() view returns(bytes32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) DomainRecipient() ([32]byte, error) {
	return _TokenMessengerWithMetadata.Contract.DomainRecipient(&_TokenMessengerWithMetadata.CallOpts)
}

// DomainRecipient is a free data retrieval call binding the contract method 0xe95f1219.
//
// Solidity: function domainRecipient() view returns(bytes32)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCallerSession) DomainRecipient() ([32]byte, error) {
	return _TokenMessengerWithMetadata.Contract.DomainRecipient(&_TokenMessengerWithMetadata.CallOpts)
}

// MessageTransmitter is a free data retrieval call binding the contract method 0x7b04c181.
//
// Solidity: function messageTransmitter() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCaller) MessageTransmitter(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TokenMessengerWithMetadata.contract.Call(opts, &out, "messageTransmitter")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MessageTransmitter is a free data retrieval call binding the contract method 0x7b04c181.
//
// Solidity: function messageTransmitter() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) MessageTransmitter() (common.Address, error) {
	return _TokenMessengerWithMetadata.Contract.MessageTransmitter(&_TokenMessengerWithMetadata.CallOpts)
}

// MessageTransmitter is a free data retrieval call binding the contract method 0x7b04c181.
//
// Solidity: function messageTransmitter() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCallerSession) MessageTransmitter() (common.Address, error) {
	return _TokenMessengerWithMetadata.Contract.MessageTransmitter(&_TokenMessengerWithMetadata.CallOpts)
}

// TokenMessenger is a free data retrieval call binding the contract method 0x46117830.
//
// Solidity: function tokenMessenger() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCaller) TokenMessenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TokenMessengerWithMetadata.contract.Call(opts, &out, "tokenMessenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// TokenMessenger is a free data retrieval call binding the contract method 0x46117830.
//
// Solidity: function tokenMessenger() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) TokenMessenger() (common.Address, error) {
	return _TokenMessengerWithMetadata.Contract.TokenMessenger(&_TokenMessengerWithMetadata.CallOpts)
}

// TokenMessenger is a free data retrieval call binding the contract method 0x46117830.
//
// Solidity: function tokenMessenger() view returns(address)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataCallerSession) TokenMessenger() (common.Address, error) {
	return _TokenMessengerWithMetadata.Contract.TokenMessenger(&_TokenMessengerWithMetadata.CallOpts)
}

// DepositForBurn is a paid mutator transaction binding the contract method 0xea04e66c.
//
// Solidity: function depositForBurn(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactor) DepositForBurn(opts *bind.TransactOpts, channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.contract.Transact(opts, "depositForBurn", channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, memo)
}

// DepositForBurn is a paid mutator transaction binding the contract method 0xea04e66c.
//
// Solidity: function depositForBurn(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) DepositForBurn(channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.DepositForBurn(&_TokenMessengerWithMetadata.TransactOpts, channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, memo)
}

// DepositForBurn is a paid mutator transaction binding the contract method 0xea04e66c.
//
// Solidity: function depositForBurn(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorSession) DepositForBurn(channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.DepositForBurn(&_TokenMessengerWithMetadata.TransactOpts, channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, memo)
}

// DepositForBurnWithCaller is a paid mutator transaction binding the contract method 0xded8b709.
//
// Solidity: function depositForBurnWithCaller(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactor) DepositForBurnWithCaller(opts *bind.TransactOpts, channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.contract.Transact(opts, "depositForBurnWithCaller", channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, destinationCaller, memo)
}

// DepositForBurnWithCaller is a paid mutator transaction binding the contract method 0xded8b709.
//
// Solidity: function depositForBurnWithCaller(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) DepositForBurnWithCaller(channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.DepositForBurnWithCaller(&_TokenMessengerWithMetadata.TransactOpts, channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, destinationCaller, memo)
}

// DepositForBurnWithCaller is a paid mutator transaction binding the contract method 0xded8b709.
//
// Solidity: function depositForBurnWithCaller(uint64 channel, bytes32 destinationBech32Prefix, bytes32 destinationRecipient, uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes memo) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorSession) DepositForBurnWithCaller(channel uint64, destinationBech32Prefix [32]byte, destinationRecipient [32]byte, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, memo []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.DepositForBurnWithCaller(&_TokenMessengerWithMetadata.TransactOpts, channel, destinationBech32Prefix, destinationRecipient, amount, mintRecipient, burnToken, destinationCaller, memo)
}

// RawDepositForBurn is a paid mutator transaction binding the contract method 0xf1e788d4.
//
// Solidity: function rawDepositForBurn(uint256 amount, bytes32 mintRecipient, address burnToken, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactor) RawDepositForBurn(opts *bind.TransactOpts, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.contract.Transact(opts, "rawDepositForBurn", amount, mintRecipient, burnToken, metadata)
}

// RawDepositForBurn is a paid mutator transaction binding the contract method 0xf1e788d4.
//
// Solidity: function rawDepositForBurn(uint256 amount, bytes32 mintRecipient, address burnToken, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) RawDepositForBurn(amount *big.Int, mintRecipient [32]byte, burnToken common.Address, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.RawDepositForBurn(&_TokenMessengerWithMetadata.TransactOpts, amount, mintRecipient, burnToken, metadata)
}

// RawDepositForBurn is a paid mutator transaction binding the contract method 0xf1e788d4.
//
// Solidity: function rawDepositForBurn(uint256 amount, bytes32 mintRecipient, address burnToken, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorSession) RawDepositForBurn(amount *big.Int, mintRecipient [32]byte, burnToken common.Address, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.RawDepositForBurn(&_TokenMessengerWithMetadata.TransactOpts, amount, mintRecipient, burnToken, metadata)
}

// RawDepositForBurnWithCaller is a paid mutator transaction binding the contract method 0x042734ee.
//
// Solidity: function rawDepositForBurnWithCaller(uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactor) RawDepositForBurnWithCaller(opts *bind.TransactOpts, amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.contract.Transact(opts, "rawDepositForBurnWithCaller", amount, mintRecipient, burnToken, destinationCaller, metadata)
}

// RawDepositForBurnWithCaller is a paid mutator transaction binding the contract method 0x042734ee.
//
// Solidity: function rawDepositForBurnWithCaller(uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataSession) RawDepositForBurnWithCaller(amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.RawDepositForBurnWithCaller(&_TokenMessengerWithMetadata.TransactOpts, amount, mintRecipient, burnToken, destinationCaller, metadata)
}

// RawDepositForBurnWithCaller is a paid mutator transaction binding the contract method 0x042734ee.
//
// Solidity: function rawDepositForBurnWithCaller(uint256 amount, bytes32 mintRecipient, address burnToken, bytes32 destinationCaller, bytes metadata) returns(uint64 nonce)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataTransactorSession) RawDepositForBurnWithCaller(amount *big.Int, mintRecipient [32]byte, burnToken common.Address, destinationCaller [32]byte, metadata []byte) (*types.Transaction, error) {
	return _TokenMessengerWithMetadata.Contract.RawDepositForBurnWithCaller(&_TokenMessengerWithMetadata.TransactOpts, amount, mintRecipient, burnToken, destinationCaller, metadata)
}

// TokenMessengerWithMetadataDepositForBurnMetadataIterator is returned from FilterDepositForBurnMetadata and is used to iterate over the raw logs and unpacked data for DepositForBurnMetadata events raised by the TokenMessengerWithMetadata contract.
type TokenMessengerWithMetadataDepositForBurnMetadataIterator struct {
	Event *TokenMessengerWithMetadataDepositForBurnMetadata // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TokenMessengerWithMetadataDepositForBurnMetadataIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TokenMessengerWithMetadataDepositForBurnMetadata)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TokenMessengerWithMetadataDepositForBurnMetadata)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TokenMessengerWithMetadataDepositForBurnMetadataIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TokenMessengerWithMetadataDepositForBurnMetadataIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TokenMessengerWithMetadataDepositForBurnMetadata represents a DepositForBurnMetadata event raised by the TokenMessengerWithMetadata contract.
type TokenMessengerWithMetadataDepositForBurnMetadata struct {
	Nonce         uint64
	MetadataNonce uint64
	Metadata      []byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterDepositForBurnMetadata is a free log retrieval operation binding the contract event 0x4c307f410548552b5dcf79f57f3f1f0b2c9dca65ec77d0fd78e8d476a7754772.
//
// Solidity: event DepositForBurnMetadata(uint64 indexed nonce, uint64 indexed metadataNonce, bytes metadata)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataFilterer) FilterDepositForBurnMetadata(opts *bind.FilterOpts, nonce []uint64, metadataNonce []uint64) (*TokenMessengerWithMetadataDepositForBurnMetadataIterator, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var metadataNonceRule []interface{}
	for _, metadataNonceItem := range metadataNonce {
		metadataNonceRule = append(metadataNonceRule, metadataNonceItem)
	}

	logs, sub, err := _TokenMessengerWithMetadata.contract.FilterLogs(opts, "DepositForBurnMetadata", nonceRule, metadataNonceRule)
	if err != nil {
		return nil, err
	}
	return &TokenMessengerWithMetadataDepositForBurnMetadataIterator{contract: _TokenMessengerWithMetadata.contract, event: "DepositForBurnMetadata", logs: logs, sub: sub}, nil
}

// WatchDepositForBurnMetadata is a free log subscription operation binding the contract event 0x4c307f410548552b5dcf79f57f3f1f0b2c9dca65ec77d0fd78e8d476a7754772.
//
// Solidity: event DepositForBurnMetadata(uint64 indexed nonce, uint64 indexed metadataNonce, bytes metadata)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataFilterer) WatchDepositForBurnMetadata(opts *bind.WatchOpts, sink chan<- *TokenMessengerWithMetadataDepositForBurnMetadata, nonce []uint64, metadataNonce []uint64) (event.Subscription, error) {

	var nonceRule []interface{}
	for _, nonceItem := range nonce {
		nonceRule = append(nonceRule, nonceItem)
	}
	var metadataNonceRule []interface{}
	for _, metadataNonceItem := range metadataNonce {
		metadataNonceRule = append(metadataNonceRule, metadataNonceItem)
	}

	logs, sub, err := _TokenMessengerWithMetadata.contract.WatchLogs(opts, "DepositForBurnMetadata", nonceRule, metadataNonceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TokenMessengerWithMetadataDepositForBurnMetadata)
				if err := _TokenMessengerWithMetadata.contract.UnpackLog(event, "DepositForBurnMetadata", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDepositForBurnMetadata is a log parse operation binding the contract event 0x4c307f410548552b5dcf79f57f3f1f0b2c9dca65ec77d0fd78e8d476a7754772.
//
// Solidity: event DepositForBurnMetadata(uint64 indexed nonce, uint64 indexed metadataNonce, bytes metadata)
func (_TokenMessengerWithMetadata *TokenMessengerWithMetadataFilterer) ParseDepositForBurnMetadata(log types.Log) (*TokenMessengerWithMetadataDepositForBurnMetadata, error) {
	event := new(TokenMessengerWithMetadataDepositForBurnMetadata)
	if err := _TokenMessengerWithMetadata.contract.UnpackLog(event, "DepositForBurnMetadata", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bitcoin

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	// Blockchain is Bitcoin.
	Blockchain string = "Bitcoin"

	// MainnetNetwork is the value of the network
	// in MainnetNetworkIdentifier.
	MainnetNetwork string = "Mainnet"

	// TestnetNetwork is the value of the network
	// in TestnetNetworkIdentifier.
	TestnetNetwork string = "Testnet3"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 8

	// SatoshisInBitcoin is the number of
	// Satoshis in 1 BTC (10^8).
	SatoshisInBitcoin = 100000000

	// InputOpType is used to describe
	// INPUT.
	InputOpType = "INPUT"

	// OutputOpType is used to describe
	// OUTPUT.
	OutputOpType = "OUTPUT"

	// CoinbaseOpType is used to describe
	// Coinbase.
	CoinbaseOpType = "COINBASE"

	// SuccessStatus is the status of all
	// Bitcoin operations because anything
	// on-chain is considered successful.
	SuccessStatus = "SUCCESS"

	// SkippedStatus is the status of all
	// operations that are skipped because
	// of BIP-30. You can read more about these
	// types of operations in BIP-30.
	SkippedStatus = "SKIPPED"

	// TransactionHashLength is the length
	// of any transaction hash in Bitcoin.
	TransactionHashLength = 64

	// NullData is returned by bitcoind
	// as the ScriptPubKey.Type for OP_RETURN
	// locking scripts.
	NullData = "nulldata"
)

// Fee estimate constants
// Source: https://bitcoinops.org/en/tools/calc-size/
const (
	MinFeeRate            = float64(0.00001) // nolint:gomnd
	TransactionOverhead   = 12               // 4 version, 2 segwit flag, 1 vin, 1 vout, 4 lock time
	InputSize             = 68               // 4 prev index, 32 prev hash, 4 sequence, 1 script size, ~27 script witness
	OutputOverhead        = 9                // 8 value, 1 script size
	P2PKHScriptPubkeySize = 25               // P2PKH size
)

// CreateMainNetParams is a function to override default mainnet settings with address prefixes
func CreateMainNetParams() (*chaincfg.Params) {
	params := &chaincfg.MainNetParams
	params.PubKeyHashAddrID = 0
	params.ScriptHashAddrID = 5
	params.Bech32HRPSegwit = "bc"
	return params
}

// CreateTestNet3Params is a function to override default testnet settings with address prefixes
func CreateTestNet3Params() (*chaincfg.Params) {
	params := &chaincfg.TestNet3Params
	params.PubKeyHashAddrID = 111
	params.ScriptHashAddrID = 196
	params.Bech32HRPSegwit = "tb"
	return params
}

var (
	// MainnetGenesisBlockIdentifier is the genesis block for mainnet.
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash: "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
	}

	// MainnetParams are the params for mainnet.
	MainnetParams = CreateMainNetParams()

	// MainnetCurrency is the *types.Currency for mainnet.
	MainnetCurrency = &types.Currency{
		Symbol:   "BTC",
		Decimals: Decimals,
	}
	
	// TestnetGenesisBlockIdentifier is the genesis block for testnet.
	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash: "000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943",
	}

	// TestnetParams are the params for testnet.
	TestnetParams = CreateTestNet3Params()

	// TestnetCurrency is the *types.Currency for testnet.
	TestnetCurrency = &types.Currency{
		Symbol:   "tBTC",
		Decimals: Decimals,
	}
	
	// OperationTypes are all supported operation.Types.
	OperationTypes = []string{
		InputOpType,
		OutputOpType,
		CoinbaseOpType,
	}

	// OperationStatuses are all supported operation.Status.
	OperationStatuses = []*types.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
		{
			Status:     SkippedStatus,
			Successful: false,
		},
	}
)

// ScriptPubKey is a script placed on the output operations
// of a Bitcoin transaction that must be satisfied to spend
// the output.
type ScriptPubKey struct {
	ASM          string   `json:"asm"`
	Hex          string   `json:"hex"`
	RequiredSigs int64    `json:"reqSigs,omitempty"`
	Type         string   `json:"type"`
	Addresses    []string `json:"addresses,omitempty"`
}

// ScriptSig is a script on the input operations of a
// Bitcoin transaction that satisfies the ScriptPubKey
// on an output being spent.
type ScriptSig struct {
	ASM string `json:"asm"`
	Hex string `json:"hex"`
}

// BlockchainInfo is information about the Bitcoin network.
// This struct only contains the information necessary for
// this implementation.
type BlockchainInfo struct {
	Chain         string `json:"chain"`
	Blocks        int64  `json:"blocks"`
	BestBlockHash string `json:"bestblockhash"`
}

// PeerInfo is a collection of relevant info about a particular peer.
type PeerInfo struct {
	Addr           string `json:"addr"`
	Version        int64  `json:"version"`
	SubVer         string `json:"subver"`
	StartingHeight int64  `json:"startingheight"`
	RelayTxes      bool   `json:"relaytxes"`
	LastSend       int64  `json:"lastsend"`
	LastRecv       int64  `json:"lastrecv"`
	BanScore       int64  `json:"banscore"`
	SyncedBlocks   int64  `json:"synced_blocks"`
	SyncedHeaders  int64  `json:"synced_headers"`
}

// Block is a raw Bitcoin block (with verbosity == 2).
type Block struct {
	Hash              string  `json:"hash"`
	Height            int64   `json:"height"`
	PreviousBlockHash string  `json:"previousblockhash"`
	Time              int64   `json:"time"`
	MedianTime        int64   `json:"mediantime"`
	Nonce             int64   `json:"nonce"`
	MerkleRoot        string  `json:"merkleroot"`
	Version           int32   `json:"version"`
	Size              int64   `json:"size"`
	Weight            int64   `json:"weight"`
	Bits              string  `json:"bits"`
	Difficulty        float64 `json:"difficulty"`

	Txs []*Transaction `json:"tx"`
}

// Metadata returns the metadata for a block.
func (b Block) Metadata() (map[string]interface{}, error) {
	m := &BlockMetadata{
		Nonce:      b.Nonce,
		MerkleRoot: b.MerkleRoot,
		Version:    b.Version,
		Size:       b.Size,
		Weight:     b.Weight,
		MedianTime: b.MedianTime,
		Bits:       b.Bits,
		Difficulty: b.Difficulty,
	}

	return types.MarshalMap(m)
}

// BlockMetadata is a collection of useful
// metadata in a block.
type BlockMetadata struct {
	Nonce      int64   `json:"nonce,omitempty"`
	MerkleRoot string  `json:"merkleroot,omitempty"`
	Version    int32   `json:"version,omitempty"`
	Size       int64   `json:"size,omitempty"`
	Weight     int64   `json:"weight,omitempty"`
	MedianTime int64   `json:"mediantime,omitempty"`
	Bits       string  `json:"bits,omitempty"`
	Difficulty float64 `json:"difficulty,omitempty"`
}

// Transaction is a raw Bitcoin transaction.
type Transaction struct {
	Hex      string `json:"hex"`
	Hash     string `json:"txid"`
	Size     int64  `json:"size"`
	Vsize    int64  `json:"vsize"`
	Version  int32  `json:"version"`
	Locktime int64  `json:"locktime"`
	Weight   int64  `json:"weight"`

	Inputs  []*Input  `json:"vin"`
	Outputs []*Output `json:"vout"`
}

// Metadata returns the metadata for a transaction.
func (t Transaction) Metadata() (map[string]interface{}, error) {
	m := &TransactionMetadata{
		Size:     t.Size,
		Vsize:    t.Vsize,
		Version:  t.Version,
		Locktime: t.Locktime,
		Weight:   t.Weight,
	}

	return types.MarshalMap(m)
}

// TransactionMetadata is a collection of useful
// metadata in a transaction.
type TransactionMetadata struct {
	Size     int64 `json:"size,omitempty"`
	Vsize    int64 `json:"vsize,omitempty"`
	Version  int32 `json:"version,omitempty"`
	Locktime int64 `json:"locktime,omitempty"`
	Weight   int64 `json:"weight,omitempty"`
}

// Input is a raw input in a Bitcoin transaction.
type Input struct {
	TxHash      string     `json:"txid"`
	Vout        int64      `json:"vout"`
	ScriptSig   *ScriptSig `json:"scriptSig"`
	Sequence    int64      `json:"sequence"`
	TxInWitness []string   `json:"txinwitness"`

	// Relevant when the input is the coinbase input
	Coinbase string `json:"coinbase"`
}

// Metadata returns the metadata for an input.
func (i Input) Metadata() (map[string]interface{}, error) {
	m := &OperationMetadata{
		ScriptSig:   i.ScriptSig,
		Sequence:    i.Sequence,
		TxInWitness: i.TxInWitness,
		Coinbase:    i.Coinbase,
	}

	return types.MarshalMap(m)
}

// Output is a raw output in a Bitcoin transaction.
type Output struct {
	Value        float64       `json:"value"`
	Index        int64         `json:"n"`
	ScriptPubKey *ScriptPubKey `json:"scriptPubKey"`
}

// Metadata returns the metadata for an output.
func (o Output) Metadata() (map[string]interface{}, error) {
	m := &OperationMetadata{
		ScriptPubKey: o.ScriptPubKey,
	}

	return types.MarshalMap(m)
}

// OperationMetadata is a collection of useful
// metadata from Bitcoin inputs and outputs.
type OperationMetadata struct {
	// Coinbase Metadata
	Coinbase string `json:"coinbase,omitempty"`

	// Input Metadata
	ScriptSig   *ScriptSig `json:"scriptsig,omitempty"`
	Sequence    int64      `json:"sequence,omitempty"`
	TxInWitness []string   `json:"txinwitness,omitempty"`

	// Output Metadata
	ScriptPubKey *ScriptPubKey `json:"scriptPubKey,omitempty"`
}

// request represents the JSON-RPC request body
type request struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

func (r request) GetVersion() string       { return r.JSONRPC }
func (r request) GetID() int               { return r.ID }
func (r request) GetMethod() string        { return r.Method }
func (r request) GetParams() []interface{} { return r.Params }

// Responses

// jSONRPCResponse represents an interface for generic JSON-RPC responses
type jSONRPCResponse interface {
	Err() error
}

type responseError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

// BlockResponse is the response body for `getblock` requests
type blockResponse struct {
	Result *Block         `json:"result"`
	Error  *responseError `json:"error"`
}

func (b blockResponse) Err() error {
	if b.Error == nil {
		return nil
	}

	if b.Error.Code == blockNotFoundErrCode {
		return ErrBlockNotFound
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		b.Error.Code,
		b.Error.Message,
	)
}

type pruneBlockchainResponse struct {
	Result int64          `json:"result"`
	Error  *responseError `json:"error"`
}

func (p pruneBlockchainResponse) Err() error {
	if p.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		p.Error.Code,
		p.Error.Message,
	)
}

type blockchainInfoResponse struct {
	Result *BlockchainInfo `json:"result"`
	Error  *responseError  `json:"error"`
}

func (b blockchainInfoResponse) Err() error {
	if b.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		b.Error.Code,
		b.Error.Message,
	)
}

type peerInfoResponse struct {
	Result []*PeerInfo    `json:"result"`
	Error  *responseError `json:"error"`
}

func (p peerInfoResponse) Err() error {
	if p.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		p.Error.Code,
		p.Error.Message,
	)
}

// blockHashResponse is the response body for `getblockhash` requests
type blockHashResponse struct {
	Result string         `json:"result"`
	Error  *responseError `json:"error"`
}

func (b blockHashResponse) Err() error {
	if b.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		b.Error.Code,
		b.Error.Message,
	)
}

// sendRawTransactionResponse is the response body for `sendrawtransaction` requests
type sendRawTransactionResponse struct {
	Result string         `json:"result"`
	Error  *responseError `json:"error"`
}

func (s sendRawTransactionResponse) Err() error {
	if s.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		s.Error.Code,
		s.Error.Message,
	)
}

type suggestedFeeRate struct {
	FeeRate float64 `json:"feerate"`
}

// suggestedFeeRateResponse is the response body for `estimatesmartfee` requests
type suggestedFeeRateResponse struct {
	Result *suggestedFeeRate `json:"result"`
	Error  *responseError    `json:"error"`
}

func (s suggestedFeeRateResponse) Err() error {
	if s.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		s.Error.Code,
		s.Error.Message,
	)
}

// rawMempoolResponse is the response body for `getrawmempool` requests.
type rawMempoolResponse struct {
	Result []string       `json:"result"`
	Error  *responseError `json:"error"`
}

func (r rawMempoolResponse) Err() error {
	if r.Error == nil {
		return nil
	}

	return fmt.Errorf(
		"%w: error JSON RPC response, code: %d, message: %s",
		ErrJSONRPCError,
		r.Error.Code,
		r.Error.Message,
	)
}

// CoinIdentifier converts a tx hash and vout into
// the canonical CoinIdentifier.Identifier used in
// rosetta-bitcoin.
func CoinIdentifier(hash string, vout int64) string {
	return fmt.Sprintf("%s:%d", hash, vout)
}

// TransactionHash extracts the transaction hash
// from a CoinIdentifier.Identifier.
func TransactionHash(identifier string) string {
	vals := strings.Split(identifier, ":")
	return vals[0]
}

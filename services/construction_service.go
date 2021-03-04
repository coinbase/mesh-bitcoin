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

package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/coinbase/rosetta-defichain/configuration"
	"github.com/coinbase/rosetta-defichain/defichain"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	// bytesInKB is the number of bytes in a KB. In Defichain, this is
	// considered to be 1000.
	bytesInKb = float64(1000) // nolint:gomnd

	// defaultConfirmationTarget is the number of blocks we would
	// like our transaction to be included by.
	defaultConfirmationTarget = int64(2) // nolint:gomnd
)

// ConstructionAPIService implements the server.ConstructionAPIServicer interface.
type ConstructionAPIService struct {
	config *configuration.Configuration
	client Client
	i      Indexer
}

// NewConstructionAPIService creates a new instance of a ConstructionAPIService.
func NewConstructionAPIService(
	config *configuration.Configuration,
	client Client,
	i Indexer,
) server.ConstructionAPIServicer {
	return &ConstructionAPIService{
		config: config,
		client: client,
		i:      i,
	}
}

// ConstructionDerive implements the /construction/derive endpoint.
func (s *ConstructionAPIService) ConstructionDerive(
	ctx context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	addr, err := btcutil.NewAddressWitnessPubKeyHash(
		btcutil.Hash160(request.PublicKey.Bytes),
		s.config.Params,
	)
	if err != nil {
		return nil, wrapErr(ErrUnableToDerive, err)
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: addr.EncodeAddress(),
		},
	}, nil
}

// estimateSize returns the estimated size of a transaction in vBytes.
func (s *ConstructionAPIService) estimateSize(operations []*types.Operation) float64 {
	size := defichain.TransactionOverhead
	for _, operation := range operations {
		switch operation.Type {
		case defichain.InputOpType:
			size += defichain.InputSize
		case defichain.OutputOpType:
			size += defichain.OutputOverhead
			addr, err := btcutil.DecodeAddress(operation.Account.Address, s.config.Params)
			if err != nil {
				size += defichain.P2PKHScriptPubkeySize
				continue
			}

			script, err := txscript.PayToAddrScript(addr)
			if err != nil {
				size += defichain.P2PKHScriptPubkeySize
				continue
			}

			size += len(script)
		}
	}

	return float64(size)
}

// ConstructionPreprocess implements the /construction/preprocess
// endpoint.
func (s *ConstructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: defichain.InputOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: s.config.Currency,
				},
				CoinAction:   types.CoinSpent,
				AllowRepeats: true,
			},
		},
	}

	matches, err := parser.MatchOperations(descriptions, request.Operations)
	if err != nil {
		return nil, wrapErr(ErrUnclearIntent, err)
	}

	coins := make([]*types.Coin, len(matches[0].Operations))
	for i, input := range matches[0].Operations {
		if input.CoinChange == nil {
			return nil, wrapErr(ErrUnclearIntent, errors.New("CoinChange cannot be nil"))
		}

		coins[i] = &types.Coin{
			CoinIdentifier: input.CoinChange.CoinIdentifier,
			Amount:         input.Amount,
		}
	}

	options, err := types.MarshalMap(&preprocessOptions{
		Coins:         coins,
		EstimatedSize: s.estimateSize(request.Operations),
		FeeMultiplier: request.SuggestedFeeMultiplier,
	})
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: options,
	}, nil
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	var options preprocessOptions
	if err := types.UnmarshalMap(request.Options, &options); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	// Determine feePerKB and ensure it is not below the minimum fee
	// relay rate.
	feePerKB, err := s.client.SuggestedFeeRate(ctx, defaultConfirmationTarget)
	if err != nil {
		return nil, wrapErr(ErrCouldNotGetFeeRate, err)
	}
	if options.FeeMultiplier != nil {
		feePerKB *= *options.FeeMultiplier
	}
	if feePerKB < defichain.MinFeeRate {
		feePerKB = defichain.MinFeeRate
	}

	// Calculated the estimated fee in Satoshis
	satoshisPerB := (feePerKB * float64(defichain.SatoshisInDFI)) / bytesInKb
	estimatedFee := satoshisPerB * options.EstimatedSize
	suggestedFee := &types.Amount{
		Value:    fmt.Sprintf("%d", int64(estimatedFee)),
		Currency: s.config.Currency,
	}

	scripts, err := s.i.GetScriptPubKeys(ctx, options.Coins)
	if err != nil {
		return nil, wrapErr(ErrScriptPubKeysMissing, err)
	}

	metadata, err := types.MarshalMap(&constructionMetadata{ScriptPubKeys: scripts})
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionMetadataResponse{
		Metadata:     metadata,
		SuggestedFee: []*types.Amount{suggestedFee},
	}, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *ConstructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: defichain.InputOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: s.config.Currency,
				},
				AllowRepeats: true,
				CoinAction:   types.CoinSpent,
			},
			{
				Type: defichain.OutputOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.PositiveAmountSign,
					Currency: s.config.Currency,
				},
				AllowRepeats: true,
			},
		},
		ErrUnmatched: true,
	}

	matches, err := parser.MatchOperations(descriptions, request.Operations)
	if err != nil {
		return nil, wrapErr(ErrUnclearIntent, err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	for _, input := range matches[0].Operations {
		if input.CoinChange == nil {
			return nil, wrapErr(ErrUnclearIntent, errors.New("CoinChange cannot be nil"))
		}

		transactionHash, index, err := defichain.ParseCoinIdentifier(input.CoinChange.CoinIdentifier)
		if err != nil {
			return nil, wrapErr(ErrInvalidCoin, err)
		}

		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *transactionHash,
				Index: index,
			},
			SignatureScript: nil,
			Sequence:        wire.MaxTxInSequenceNum,
		})
	}

	for i, output := range matches[1].Operations {
		addr, err := btcutil.DecodeAddress(output.Account.Address, s.config.Params)
		if err != nil {
			return nil, wrapErr(ErrUnableToDecodeAddress, fmt.Errorf(
				"%w unable to decode address %s",
				err,
				output.Account.Address,
			),
			)
		}

		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to construct payToAddrScript", err),
			)
		}

		tx.AddTxOut(&wire.TxOut{
			Value:    matches[1].Amounts[i].Int64(),
			PkScript: pkScript,
		})
	}

	// Create Signing Payloads (must be done after entire tx is constructed
	// or hash will not be correct).
	inputAmounts := make([]string, len(tx.TxIn))
	inputAddresses := make([]string, len(tx.TxIn))
	payloads := make([]*types.SigningPayload, len(tx.TxIn))
	var metadata constructionMetadata
	if err := types.UnmarshalMap(request.Metadata, &metadata); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	for i := range tx.TxIn {
		address := matches[0].Operations[i].Account.Address
		script, err := hex.DecodeString(metadata.ScriptPubKeys[i].Hex)
		if err != nil {
			return nil, wrapErr(ErrUnableToDecodeScriptPubKey, err)
		}

		class, _, err := defichain.ParseSingleAddress(s.config.Params, script)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to parse address for utxo %d", err, i),
			)
		}

		inputAddresses[i] = address
		inputAmounts[i] = matches[0].Amounts[i].String()
		absAmount := new(big.Int).Abs(matches[0].Amounts[i]).Int64()

		switch class {
		case txscript.WitnessV0PubKeyHashTy:
			hash, err := txscript.CalcWitnessSigHash(
				script,
				txscript.NewTxSigHashes(tx),
				txscript.SigHashAll,
				tx,
				i,
				absAmount,
			)
			if err != nil {
				return nil, wrapErr(ErrUnableToCalculateSignatureHash, err)
			}

			payloads[i] = &types.SigningPayload{
				AccountIdentifier: &types.AccountIdentifier{
					Address: address,
				},
				Bytes:         hash,
				SignatureType: types.Ecdsa,
			}
		default:
			return nil, wrapErr(
				ErrUnsupportedScriptType,
				fmt.Errorf("unupported script type: %s", class),
			)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	rawTx, err := json.Marshal(&unsignedTransaction{
		Transaction:    hex.EncodeToString(buf.Bytes()),
		ScriptPubKeys:  metadata.ScriptPubKeys,
		InputAmounts:   inputAmounts,
		InputAddresses: inputAddresses,
	})
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: hex.EncodeToString(rawTx),
		Payloads:            payloads,
	}, nil
}

func normalizeSignature(signature []byte) []byte {
	sig := btcec.Signature{ // signature is in form of R || S
		R: new(big.Int).SetBytes(signature[:32]),
		S: new(big.Int).SetBytes(signature[32:64]),
	}

	return append(sig.Serialize(), byte(txscript.SigHashAll))
}

// ConstructionCombine implements the /construction/combine
// endpoint.
func (s *ConstructionAPIService) ConstructionCombine(
	ctx context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	decodedTx, err := hex.DecodeString(request.UnsignedTransaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w transaction cannot be decoded", err),
		)
	}

	var unsigned unsignedTransaction
	if err := json.Unmarshal(decodedTx, &unsigned); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to unmarshal defichain transaction", err),
		)
	}

	decodedCoreTx, err := hex.DecodeString(unsigned.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w transaction cannot be decoded", err),
		)
	}

	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(decodedCoreTx)); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to deserialize tx", err),
		)
	}

	for i := range tx.TxIn {
		decodedScript, err := hex.DecodeString(unsigned.ScriptPubKeys[i].Hex)
		if err != nil {
			return nil, wrapErr(ErrUnableToDecodeScriptPubKey, err)
		}

		class, _, err := defichain.ParseSingleAddress(s.config.Params, decodedScript)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to parse address for script", err),
			)
		}

		pkData := request.Signatures[i].PublicKey.Bytes
		fullsig := normalizeSignature(request.Signatures[i].Bytes)

		switch class {
		case txscript.WitnessV0PubKeyHashTy:
			tx.TxIn[i].Witness = wire.TxWitness{fullsig, pkData}
		default:
			return nil, wrapErr(
				ErrUnsupportedScriptType,
				fmt.Errorf("unupported script type: %s", class),
			)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	if err := tx.Serialize(buf); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, fmt.Errorf("%w serialize tx", err))
	}

	rawTx, err := json.Marshal(&signedTransaction{
		Transaction:  hex.EncodeToString(buf.Bytes()),
		InputAmounts: unsigned.InputAmounts,
	})
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to serialize signed tx", err),
		)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: hex.EncodeToString(rawTx),
	}, nil
}

// ConstructionHash implements the /construction/hash endpoint.
func (s *ConstructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	decodedTx, err := hex.DecodeString(request.SignedTransaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w signed transaction cannot be decoded", err),
		)
	}

	var signed signedTransaction
	if err := json.Unmarshal(decodedTx, &signed); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to unmarshal signed defichain transaction", err),
		)
	}

	bytesTx, err := hex.DecodeString(signed.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to decode hex transaction", err),
		)
	}

	tx, err := btcutil.NewTxFromBytes(bytesTx)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to parse transaction", err),
		)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: tx.Hash().String(),
		},
	}, nil
}

func (s *ConstructionAPIService) parseUnsignedTransaction(
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	decodedTx, err := hex.DecodeString(request.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w transaction cannot be decoded", err),
		)
	}

	var unsigned unsignedTransaction
	if err := json.Unmarshal(decodedTx, &unsigned); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to unmarshal defichain transaction", err),
		)
	}

	decodedCoreTx, err := hex.DecodeString(unsigned.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w transaction cannot be decoded", err),
		)
	}

	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(decodedCoreTx)); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to deserialize tx", err),
		)
	}

	ops := []*types.Operation{}
	for i, input := range tx.TxIn {
		networkIndex := int64(i)
		ops = append(ops, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        int64(len(ops)),
				NetworkIndex: &networkIndex,
			},
			Type: defichain.InputOpType,
			Account: &types.AccountIdentifier{
				Address: unsigned.InputAddresses[i],
			},
			Amount: &types.Amount{
				Value:    unsigned.InputAmounts[i],
				Currency: s.config.Currency,
			},
			CoinChange: &types.CoinChange{
				CoinAction: types.CoinSpent,
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: fmt.Sprintf(
						"%s:%d",
						input.PreviousOutPoint.Hash.String(),
						input.PreviousOutPoint.Index,
					),
				},
			},
		})
	}

	for i, output := range tx.TxOut {
		networkIndex := int64(i)
		_, addr, err := defichain.ParseSingleAddress(s.config.Params, output.PkScript)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to parse output address", err),
			)
		}

		ops = append(ops, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        int64(len(ops)),
				NetworkIndex: &networkIndex,
			},
			Type: defichain.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: addr.String(),
			},
			Amount: &types.Amount{
				Value:    strconv.FormatInt(output.Value, 10),
				Currency: s.config.Currency,
			},
		})
	}

	return &types.ConstructionParseResponse{
		Operations:               ops,
		AccountIdentifierSigners: []*types.AccountIdentifier{},
	}, nil
}

func (s *ConstructionAPIService) parseSignedTransaction(
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	decodedTx, err := hex.DecodeString(request.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w signed transaction cannot be decoded", err),
		)
	}

	var signed signedTransaction
	if err := json.Unmarshal(decodedTx, &signed); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to unmarshal signed defichain transaction", err),
		)
	}

	serializedTx, err := hex.DecodeString(signed.Transaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to decode hex transaction", err),
		)
	}

	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(serializedTx)); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to decode msgTx", err),
		)
	}

	ops := []*types.Operation{}
	signers := []*types.AccountIdentifier{}
	for i, input := range tx.TxIn {
		pkScript, err := txscript.ComputePkScript(input.SignatureScript, input.Witness)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToComputePkScript,
				fmt.Errorf("%w: unable to compute pk script", err),
			)
		}

		_, addr, err := defichain.ParseSingleAddress(s.config.Params, pkScript.Script())
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to decode address", err),
			)
		}

		networkIndex := int64(i)
		signers = append(signers, &types.AccountIdentifier{
			Address: addr.EncodeAddress(),
		})
		ops = append(ops, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        int64(len(ops)),
				NetworkIndex: &networkIndex,
			},
			Type: defichain.InputOpType,
			Account: &types.AccountIdentifier{
				Address: addr.EncodeAddress(),
			},
			Amount: &types.Amount{
				Value:    signed.InputAmounts[i],
				Currency: s.config.Currency,
			},
			CoinChange: &types.CoinChange{
				CoinAction: types.CoinSpent,
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: fmt.Sprintf(
						"%s:%d",
						input.PreviousOutPoint.Hash.String(),
						input.PreviousOutPoint.Index,
					),
				},
			},
		})
	}

	for i, output := range tx.TxOut {
		networkIndex := int64(i)
		_, addr, err := defichain.ParseSingleAddress(s.config.Params, output.PkScript)
		if err != nil {
			return nil, wrapErr(
				ErrUnableToDecodeAddress,
				fmt.Errorf("%w unable to parse output address", err),
			)
		}

		ops = append(ops, &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        int64(len(ops)),
				NetworkIndex: &networkIndex,
			},
			Type: defichain.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: addr.String(),
			},
			Amount: &types.Amount{
				Value:    strconv.FormatInt(output.Value, 10),
				Currency: s.config.Currency,
			},
		})
	}

	return &types.ConstructionParseResponse{
		Operations:               ops,
		AccountIdentifierSigners: signers,
	}, nil
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *ConstructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	if request.Signed {
		return s.parseSignedTransaction(request)
	}

	return s.parseUnsignedTransaction(request)
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	decodedTx, err := hex.DecodeString(request.SignedTransaction)
	if err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w signed transaction cannot be decoded", err),
		)
	}

	var signed signedTransaction
	if err := json.Unmarshal(decodedTx, &signed); err != nil {
		return nil, wrapErr(
			ErrUnableToParseIntermediateResult,
			fmt.Errorf("%w unable to unmarshal signed defichain transaction", err),
		)
	}

	txHash, err := s.client.SendRawTransaction(ctx, signed.Transaction)
	if err != nil {
		return nil, wrapErr(ErrDefid, fmt.Errorf("%w unable to submit transaction", err))
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txHash,
		},
	}, nil
}

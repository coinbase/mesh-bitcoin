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
	"context"
	"encoding/hex"
	"testing"

	"github.com/coinbase/rosetta-bitcoin/bitcoin"
	"github.com/coinbase/rosetta-bitcoin/configuration"
	mocks "github.com/coinbase/rosetta-bitcoin/mocks/services"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func forceHexDecode(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("could not decode hex %s", s)
	}

	return b
}

func forceMarshalMap(t *testing.T, i interface{}) map[string]interface{} {
	m, err := types.MarshalMap(i)
	if err != nil {
		t.Fatalf("could not marshal map %s", types.PrintStruct(i))
	}

	return m
}

func TestConstructionService(t *testing.T) {
	networkIdentifier = &types.NetworkIdentifier{
		Network:    bitcoin.TestnetNetwork,
		Blockchain: bitcoin.Blockchain,
	}

	cfg := &configuration.Configuration{
		Mode:     configuration.Online,
		Network:  networkIdentifier,
		Params:   bitcoin.TestnetParams,
		Currency: bitcoin.TestnetCurrency,
	}

	mockIndexer := &mocks.Indexer{}
	mockClient := &mocks.Client{}
	servicer := NewConstructionAPIService(cfg, mockClient, mockIndexer)
	ctx := context.Background()

	// Test Derive
	publicKey := &types.PublicKey{
		Bytes: forceHexDecode(
			t,
			"0325c9a4252789b31dbb3454ec647e9516e7c596bcde2bd5da71a60fab8644e438",
		),
		CurveType: types.Secp256k1,
	}
	deriveResponse, err := servicer.ConstructionDerive(ctx, &types.ConstructionDeriveRequest{
		NetworkIdentifier: networkIdentifier,
		PublicKey:         publicKey,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: "tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm",
		},
	}, deriveResponse)

	// Test Preprocess
	ops := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type: bitcoin.InputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm",
			},
			Amount: &types.Amount{
				Value:    "-1000000",
				Currency: bitcoin.TestnetCurrency,
			},
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: "b14157a5c50503c8cd202a173613dd27e0027343c3d50cf85852dd020bf59c7f:1",
				},
				CoinAction: types.CoinSpent,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			Type: bitcoin.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1q3r8xjf0c2yazxnq9ey3wayelygfjxpfqjvj5v7",
			},
			Amount: &types.Amount{
				Value:    "954843",
				Currency: bitcoin.TestnetCurrency,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 2,
			},
			Type: bitcoin.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1qjsrjvk2ug872pdypp33fjxke62y7awpgefr6ua",
			},
			Amount: &types.Amount{
				Value:    "44657",
				Currency: bitcoin.TestnetCurrency,
			},
		},
	}
	feeMultiplier := float64(0.75)
	preprocessResponse, err := servicer.ConstructionPreprocess(
		ctx,
		&types.ConstructionPreprocessRequest{
			NetworkIdentifier:      networkIdentifier,
			Operations:             ops,
			SuggestedFeeMultiplier: &feeMultiplier,
		},
	)
	assert.Nil(t, err)
	options := &preprocessOptions{
		Coins: []*types.Coin{
			{
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: "b14157a5c50503c8cd202a173613dd27e0027343c3d50cf85852dd020bf59c7f:1",
				},
				Amount: &types.Amount{
					Value:    "-1000000",
					Currency: bitcoin.TestnetCurrency,
				},
			},
		},
		EstimatedSize: 142,
		FeeMultiplier: &feeMultiplier,
	}
	assert.Equal(t, &types.ConstructionPreprocessResponse{
		Options: forceMarshalMap(t, options),
	}, preprocessResponse)

	// Test Metadata
	metadata := &constructionMetadata{
		ScriptPubKeys: []*bitcoin.ScriptPubKey{
			{
				ASM:          "0 c005b00ad075d30b89a7b65b7dad8899ba6a9c55",
				Hex:          "0014c005b00ad075d30b89a7b65b7dad8899ba6a9c55",
				RequiredSigs: 1,
				Type:         "witness_v0_keyhash",
				Addresses: []string{
					"tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm",
				},
			},
		},
	}

	// Normal Fee
	mockIndexer.On(
		"GetScriptPubKeys",
		ctx,
		options.Coins,
	).Return(
		metadata.ScriptPubKeys,
		nil,
	).Once()
	mockClient.On(
		"SuggestedFeeRate",
		ctx,
		defaultConfirmationTarget,
	).Return(
		bitcoin.MinFeeRate*10,
		nil,
	).Once()
	metadataResponse, err := servicer.ConstructionMetadata(ctx, &types.ConstructionMetadataRequest{
		NetworkIdentifier: networkIdentifier,
		Options:           forceMarshalMap(t, options),
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionMetadataResponse{
		Metadata: forceMarshalMap(t, metadata),
		SuggestedFee: []*types.Amount{
			{
				Value:    "1065", // 1,420 * 0.75
				Currency: bitcoin.TestnetCurrency,
			},
		},
	}, metadataResponse)

	// Low Fee
	mockIndexer.On(
		"GetScriptPubKeys",
		ctx,
		options.Coins,
	).Return(
		metadata.ScriptPubKeys,
		nil,
	).Once()
	mockClient.On(
		"SuggestedFeeRate",
		ctx,
		defaultConfirmationTarget,
	).Return(
		bitcoin.MinFeeRate,
		nil,
	).Once()
	metadataResponse, err = servicer.ConstructionMetadata(ctx, &types.ConstructionMetadataRequest{
		NetworkIdentifier: networkIdentifier,
		Options:           forceMarshalMap(t, options),
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionMetadataResponse{
		Metadata: forceMarshalMap(t, metadata),
		SuggestedFee: []*types.Amount{
			{
				Value:    "142", // we don't go below minimum fee rate
				Currency: bitcoin.TestnetCurrency,
			},
		},
	}, metadataResponse)

	// Test Payloads
	unsignedRaw := "7b227472616e73616374696f6e223a2230313030303030303031376639636635306230326464353235386638306364356333343337333032653032376464313333363137326132306364633830333035633561353537343162313031303030303030303066666666666666663032646239313065303030303030303030303136303031343838636536393235663835313361323334633035633932326565393333663232313332333035323037316165303030303030303030303030313630303134393430373236353935633431666361306234383130633632393931616439643238396565623832383030303030303030222c227363726970745075624b657973223a5b7b2261736d223a22302063303035623030616430373564333062383961376236356237646164383839396261366139633535222c22686578223a223030313463303035623030616430373564333062383961376236356237646164383839396261366139633535222c2272657153696773223a312c2274797065223a227769746e6573735f76305f6b657968617368222c22616464726573736573223a5b227462317163717a6d717a6b7377686673687a64386b6564686d7476676e78617834387a34666b6c68766d225d7d5d2c22696e7075745f616d6f756e7473223a5b222d31303030303030225d2c22696e7075745f616464726573736573223a5b227462317163717a6d717a6b7377686673687a64386b6564686d7476676e78617834387a34666b6c68766d225d7d" // nolint
	payloadsResponse, err := servicer.ConstructionPayloads(ctx, &types.ConstructionPayloadsRequest{
		NetworkIdentifier: networkIdentifier,
		Operations:        ops,
		Metadata:          forceMarshalMap(t, metadata),
	})
	val0 := int64(0)
	val1 := int64(1)
	parseOps := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        0,
				NetworkIndex: &val0,
			},
			Type: bitcoin.InputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm",
			},
			Amount: &types.Amount{
				Value:    "-1000000",
				Currency: bitcoin.TestnetCurrency,
			},
			CoinChange: &types.CoinChange{
				CoinIdentifier: &types.CoinIdentifier{
					Identifier: "b14157a5c50503c8cd202a173613dd27e0027343c3d50cf85852dd020bf59c7f:1",
				},
				CoinAction: types.CoinSpent,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        1,
				NetworkIndex: &val0,
			},
			Type: bitcoin.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1q3r8xjf0c2yazxnq9ey3wayelygfjxpfqjvj5v7",
			},
			Amount: &types.Amount{
				Value:    "954843",
				Currency: bitcoin.TestnetCurrency,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index:        2,
				NetworkIndex: &val1,
			},
			Type: bitcoin.OutputOpType,
			Account: &types.AccountIdentifier{
				Address: "tb1qjsrjvk2ug872pdypp33fjxke62y7awpgefr6ua",
			},
			Amount: &types.Amount{
				Value:    "44657",
				Currency: bitcoin.TestnetCurrency,
			},
		},
	}

	assert.Nil(t, err)
	signingPayload := &types.SigningPayload{
		Bytes: forceHexDecode(
			t,
			"7b98f8b77fa6ef34044f320073118033afdffbd3fd3f8423889d9e5953ff4a30",
		),
		AccountIdentifier: &types.AccountIdentifier{
			Address: "tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm",
		},
		SignatureType: types.Ecdsa,
	}
	assert.Equal(t, &types.ConstructionPayloadsResponse{
		UnsignedTransaction: unsignedRaw,
		Payloads:            []*types.SigningPayload{signingPayload},
	}, payloadsResponse)

	// Test Parse Unsigned
	parseUnsignedResponse, err := servicer.ConstructionParse(ctx, &types.ConstructionParseRequest{
		NetworkIdentifier: networkIdentifier,
		Signed:            false,
		Transaction:       unsignedRaw,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionParseResponse{
		Operations:               parseOps,
		AccountIdentifierSigners: []*types.AccountIdentifier{},
	}, parseUnsignedResponse)

	// Test Combine
	signedRaw := "7b227472616e73616374696f6e223a22303130303030303030303031303137663963663530623032646435323538663830636435633334333733303265303237646431333336313732613230636463383033303563356135353734316231303130303030303030306666666666666666303264623931306530303030303030303030313630303134383863653639323566383531336132333463303563393232656539333366323231333233303532303731616530303030303030303030303031363030313439343037323635393563343166636130623438313063363239393161643964323839656562383238303234373330343430323230323538373665633862396635316433343361356135366163353439633063383238303035656634356562653964613136366462363435633039313537323233663032323034636430386237323738613838383961383131333539313562636531306431656633626239326232313766383161306465376537396666623364666436616335303132313033323563396134323532373839623331646262333435346563363437653935313665376335393662636465326264356461373161363066616238363434653433383030303030303030222c22696e7075745f616d6f756e7473223a5b222d31303030303030225d7d" // nolint
	combineResponse, err := servicer.ConstructionCombine(ctx, &types.ConstructionCombineRequest{
		NetworkIdentifier:   networkIdentifier,
		UnsignedTransaction: unsignedRaw,
		Signatures: []*types.Signature{
			{
				Bytes: forceHexDecode(
					t,
					"25876ec8b9f51d343a5a56ac549c0c828005ef45ebe9da166db645c09157223f4cd08b7278a8889a81135915bce10d1ef3bb92b217f81a0de7e79ffb3dfd6ac5", // nolint
				),
				SigningPayload: signingPayload,
				PublicKey:      publicKey,
				SignatureType:  types.Ecdsa,
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionCombineResponse{
		SignedTransaction: signedRaw,
	}, combineResponse)

	// Test Parse Signed
	parseSignedResponse, err := servicer.ConstructionParse(ctx, &types.ConstructionParseRequest{
		NetworkIdentifier: networkIdentifier,
		Signed:            true,
		Transaction:       signedRaw,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.ConstructionParseResponse{
		Operations: parseOps,
		AccountIdentifierSigners: []*types.AccountIdentifier{
			{Address: "tb1qcqzmqzkswhfshzd8kedhmtvgnxax48z4fklhvm"},
		},
	}, parseSignedResponse)

	// Test Hash
	transactionIdentifier := &types.TransactionIdentifier{
		Hash: "6d87ad0e26025128f5a8357fa423b340cbcffb9703f79f432f5520fca59cd20b",
	}
	hashResponse, err := servicer.ConstructionHash(ctx, &types.ConstructionHashRequest{
		NetworkIdentifier: networkIdentifier,
		SignedTransaction: signedRaw,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.TransactionIdentifierResponse{
		TransactionIdentifier: transactionIdentifier,
	}, hashResponse)

	// Test Submit
	bitcoinTransaction := "010000000001017f9cf50b02dd5258f80cd5c3437302e027dd1336172a20cdc80305c5a55741b10100000000ffffffff02db910e000000000016001488ce6925f8513a234c05c922ee933f221323052071ae000000000000160014940726595c41fca0b4810c62991ad9d289eeb82802473044022025876ec8b9f51d343a5a56ac549c0c828005ef45ebe9da166db645c09157223f02204cd08b7278a8889a81135915bce10d1ef3bb92b217f81a0de7e79ffb3dfd6ac501210325c9a4252789b31dbb3454ec647e9516e7c596bcde2bd5da71a60fab8644e43800000000" // nolint
	mockClient.On(
		"SendRawTransaction",
		ctx,
		bitcoinTransaction,
	).Return(
		transactionIdentifier.Hash,
		nil,
	)
	submitResponse, err := servicer.ConstructionSubmit(ctx, &types.ConstructionSubmitRequest{
		NetworkIdentifier: networkIdentifier,
		SignedTransaction: signedRaw,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.TransactionIdentifierResponse{
		TransactionIdentifier: transactionIdentifier,
	}, submitResponse)

	mockClient.AssertExpectations(t)
	mockIndexer.AssertExpectations(t)
}

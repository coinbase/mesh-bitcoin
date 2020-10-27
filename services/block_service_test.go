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
	"fmt"
	"testing"

	"github.com/coinbase/rosetta-bitcoin/configuration"
	mocks "github.com/coinbase/rosetta-bitcoin/mocks/services"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestBlockService_Offline(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.Offline,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewBlockAPIService(cfg, mockIndexer)
	ctx := context.Background()

	block, err := servicer.Block(ctx, &types.BlockRequest{})
	assert.Nil(t, block)
	assert.Equal(t, ErrUnavailableOffline.Code, err.Code)
	assert.Equal(t, ErrUnavailableOffline.Message, err.Message)

	blockTransaction, err := servicer.BlockTransaction(ctx, &types.BlockTransactionRequest{})
	assert.Nil(t, blockTransaction)
	assert.Equal(t, ErrUnavailableOffline.Code, err.Code)
	assert.Equal(t, ErrUnavailableOffline.Message, err.Message)

	mockIndexer.AssertExpectations(t)
}

func TestBlockService_Online_Inline(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.Online,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewBlockAPIService(cfg, mockIndexer)
	ctx := context.Background()

	rawBlock := &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 100,
			Hash:  "block 100",
		},
	}

	transaction := &types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: "tx1",
		},
	}

	block := &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 100,
			Hash:  "block 100",
		},
		Transactions: []*types.Transaction{
			transaction,
		},
	}

	blockResponse := &types.BlockResponse{
		Block: block,
	}

	t.Run("nil identifier", func(t *testing.T) {
		mockIndexer.On(
			"GetBlockLazy",
			ctx,
			(*types.PartialBlockIdentifier)(nil),
		).Return(
			&types.BlockResponse{
				Block: rawBlock,
				OtherTransactions: []*types.TransactionIdentifier{
					{
						Hash: "tx1",
					},
				},
			},
			nil,
		).Once()
		mockIndexer.On(
			"GetBlockTransaction",
			ctx,
			blockResponse.Block.BlockIdentifier,
			transaction.TransactionIdentifier,
		).Return(
			transaction,
			nil,
		).Once()
		b, err := servicer.Block(ctx, &types.BlockRequest{})
		assert.Nil(t, err)
		assert.Equal(t, blockResponse, b)
	})

	t.Run("populated identifier", func(t *testing.T) {
		pbIdentifier := types.ConstructPartialBlockIdentifier(block.BlockIdentifier)
		mockIndexer.On(
			"GetBlockLazy",
			ctx,
			pbIdentifier,
		).Return(
			&types.BlockResponse{
				Block: rawBlock,
				OtherTransactions: []*types.TransactionIdentifier{
					{
						Hash: "tx1",
					},
				},
			},
			nil,
		).Once()
		mockIndexer.On(
			"GetBlockTransaction",
			ctx,
			blockResponse.Block.BlockIdentifier,
			transaction.TransactionIdentifier,
		).Return(
			transaction,
			nil,
		).Once()
		b, err := servicer.Block(ctx, &types.BlockRequest{
			BlockIdentifier: pbIdentifier,
		})
		assert.Nil(t, err)
		assert.Equal(t, blockResponse, b)
	})

	mockIndexer.AssertExpectations(t)
}

func TestBlockService_Online_External(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.Online,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewBlockAPIService(cfg, mockIndexer)
	ctx := context.Background()

	blockResponse := &types.BlockResponse{
		Block: &types.Block{
			BlockIdentifier: &types.BlockIdentifier{
				Index: 100,
				Hash:  "block 100",
			},
		},
	}

	otherTxs := []*types.TransactionIdentifier{}
	for i := 0; i < 200; i++ {
		otherTxs = append(otherTxs, &types.TransactionIdentifier{
			Hash: fmt.Sprintf("tx%d", i),
		})
	}
	blockResponse.OtherTransactions = otherTxs

	mockIndexer.On(
		"GetBlockLazy",
		ctx,
		(*types.PartialBlockIdentifier)(nil),
	).Return(
		blockResponse,
		nil,
	).Once()
	b, err := servicer.Block(ctx, &types.BlockRequest{})
	assert.Nil(t, err)
	assert.Equal(t, blockResponse, b)

	for _, otherTx := range b.OtherTransactions {
		tx := &types.Transaction{
			TransactionIdentifier: otherTx,
		}
		mockIndexer.On(
			"GetBlockTransaction",
			ctx,
			blockResponse.Block.BlockIdentifier,
			otherTx,
		).Return(
			tx,
			nil,
		).Once()

		bTx, err := servicer.BlockTransaction(ctx, &types.BlockTransactionRequest{
			BlockIdentifier:       blockResponse.Block.BlockIdentifier,
			TransactionIdentifier: otherTx,
		})
		assert.Nil(t, err)
		assert.Equal(t, &types.BlockTransactionResponse{
			Transaction: tx,
		}, bTx)

	}

	mockIndexer.AssertExpectations(t)
}

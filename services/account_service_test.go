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
	"testing"

	"github.com/coinbase/rosetta-bitcoin/bitcoin"
	"github.com/coinbase/rosetta-bitcoin/configuration"
	mocks "github.com/coinbase/rosetta-bitcoin/mocks/services"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestAccountBalance_Offline(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.Offline,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewAccountAPIService(cfg, mockIndexer)
	ctx := context.Background()

	bal, err := servicer.AccountBalance(ctx, &types.AccountBalanceRequest{})
	assert.Nil(t, bal)
	assert.Equal(t, ErrUnavailableOffline.Code, err.Code)

	coins, err := servicer.AccountCoins(ctx, &types.AccountCoinsRequest{})
	assert.Nil(t, coins)
	assert.Equal(t, ErrUnavailableOffline.Code, err.Code)

	mockIndexer.AssertExpectations(t)
}

func TestAccountBalance_Online_Current(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode:     configuration.Online,
		Currency: bitcoin.MainnetCurrency,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewAccountAPIService(cfg, mockIndexer)
	ctx := context.Background()
	account := &types.AccountIdentifier{
		Address: "hello",
	}
	block := &types.BlockIdentifier{
		Index: 1000,
		Hash:  "block 1000",
	}
	amount := &types.Amount{
		Value:    "25",
		Currency: bitcoin.MainnetCurrency,
	}

	mockIndexer.On(
		"GetBalance",
		ctx,
		account,
		bitcoin.MainnetCurrency,
		(*types.PartialBlockIdentifier)(nil),
	).Return(amount, block, nil).Once()
	bal, err := servicer.AccountBalance(ctx, &types.AccountBalanceRequest{
		AccountIdentifier: account,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.AccountBalanceResponse{
		BlockIdentifier: block,
		Balances: []*types.Amount{
			amount,
		},
	}, bal)

	mockIndexer.AssertExpectations(t)
}

func TestAccountBalance_Online_Historical(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode:     configuration.Online,
		Currency: bitcoin.MainnetCurrency,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewAccountAPIService(cfg, mockIndexer)
	ctx := context.Background()
	account := &types.AccountIdentifier{
		Address: "hello",
	}
	block := &types.BlockIdentifier{
		Index: 1000,
		Hash:  "block 1000",
	}
	partialBlock := &types.PartialBlockIdentifier{
		Index: &block.Index,
	}
	amount := &types.Amount{
		Value:    "25",
		Currency: bitcoin.MainnetCurrency,
	}

	mockIndexer.On(
		"GetBalance",
		ctx,
		account,
		bitcoin.MainnetCurrency,
		partialBlock,
	).Return(amount, block, nil).Once()
	bal, err := servicer.AccountBalance(ctx, &types.AccountBalanceRequest{
		AccountIdentifier: account,
		BlockIdentifier:   partialBlock,
	})
	assert.Nil(t, err)
	assert.Equal(t, &types.AccountBalanceResponse{
		BlockIdentifier: block,
		Balances: []*types.Amount{
			amount,
		},
	}, bal)

	mockIndexer.AssertExpectations(t)
}

func TestAccountCoins_Online(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode:     configuration.Online,
		Currency: bitcoin.MainnetCurrency,
	}
	mockIndexer := &mocks.Indexer{}
	servicer := NewAccountAPIService(cfg, mockIndexer)
	ctx := context.Background()

	account := &types.AccountIdentifier{
		Address: "hello",
	}

	coins := []*types.Coin{
		{
			Amount: &types.Amount{
				Value: "10",
			},
			CoinIdentifier: &types.CoinIdentifier{
				Identifier: "coin 1",
			},
		},
		{
			Amount: &types.Amount{
				Value: "15",
			},
			CoinIdentifier: &types.CoinIdentifier{
				Identifier: "coin 2",
			},
		},
		{
			Amount: &types.Amount{
				Value: "0",
			},
			CoinIdentifier: &types.CoinIdentifier{
				Identifier: "coin 3",
			},
		},
	}
	block := &types.BlockIdentifier{
		Index: 1000,
		Hash:  "block 1000",
	}
	mockIndexer.On("GetCoins", ctx, account).Return(coins, block, nil).Once()

	bal, err := servicer.AccountCoins(ctx, &types.AccountCoinsRequest{
		AccountIdentifier: account,
	})
	assert.Nil(t, err)

	assert.Equal(t, &types.AccountCoinsResponse{
		BlockIdentifier: block,
		Coins:           coins,
	}, bal)

	mockIndexer.AssertExpectations(t)
}

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

	"github.com/coinbase/rosetta-defichain/configuration"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// AccountAPIService implements the server.AccountAPIServicer interface.
type AccountAPIService struct {
	config *configuration.Configuration
	i      Indexer
}

// NewAccountAPIService returns a new *AccountAPIService.
func NewAccountAPIService(
	config *configuration.Configuration,
	i Indexer,
) server.AccountAPIServicer {
	return &AccountAPIService{
		config: config,
		i:      i,
	}
}

// AccountBalance implements /account/balance.
func (s *AccountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	// TODO: filter balances by request currencies

	// If we are fetching a historical balance,
	// use balance storage and don't return coins.
	amount, block, err := s.i.GetBalance(
		ctx,
		request.AccountIdentifier,
		s.config.Currency,
		request.BlockIdentifier,
	)
	if err != nil {
		return nil, wrapErr(ErrUnableToGetBalance, err)
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: block,
		Balances: []*types.Amount{
			amount,
		},
	}, nil
}

// AccountCoins implements /account/coins.
func (s *AccountAPIService) AccountCoins(
	ctx context.Context,
	request *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	// TODO: filter coins by request currencies

	// TODO: support include_mempool query
	// https://github.com/coinbase/rosetta-bitcoin/issues/36#issuecomment-724992022
	// Once mempoolcoins are supported also change the bool service/types.go:MempoolCoins to true

	coins, block, err := s.i.GetCoins(ctx, request.AccountIdentifier)
	if err != nil {
		return nil, wrapErr(ErrUnableToGetCoins, err)
	}

	result := &types.AccountCoinsResponse{
		BlockIdentifier: block,
		Coins:           coins,
	}

	return result, nil
}

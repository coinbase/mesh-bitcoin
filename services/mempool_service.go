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

	"github.com/coinbase/rosetta-defichain/configuration"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// MempoolAPIService implements the server.MempoolAPIServicer interface.
type MempoolAPIService struct {
	config *configuration.Configuration
	client Client
}

// NewMempoolAPIService creates a new instance of a MempoolAPIService.
func NewMempoolAPIService(
	config *configuration.Configuration,
	client Client,
) server.MempoolAPIServicer {
	return &MempoolAPIService{
		config: config,
		client: client,
	}
}

// Mempool implements the /mempool endpoint.
func (s *MempoolAPIService) Mempool(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	mempoolTransactions, err := s.client.RawMempool(ctx)
	if err != nil {
		return nil, wrapErr(ErrDefichaind, err)
	}

	transactionIdentifiers := make([]*types.TransactionIdentifier, len(mempoolTransactions))
	for i, mempoolTransaction := range mempoolTransactions {
		transactionIdentifiers[i] = &types.TransactionIdentifier{Hash: mempoolTransaction}
	}

	return &types.MempoolResponse{
		TransactionIdentifiers: transactionIdentifiers,
	}, nil
}

// MempoolTransaction implements the /mempool/transaction endpoint.
func (s *MempoolAPIService) MempoolTransaction(
	ctx context.Context,
	request *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, wrapErr(ErrUnavailableOffline, nil)
	}

	txHash := ""
	if request != nil {
		if request.TransactionIdentifier != nil {
			txHash = request.TransactionIdentifier.Hash
		}
	}

	tx, err := s.client.GetRawTransaction(ctx, txHash, "")
	if err != nil || tx == nil || tx.Hash == "" {
		// if err != nil || tx == nil {
		return nil, wrapErr(ErrTransactionNotFound, nil)
	}
	// FIXME: delete
	fmt.Printf("/mempool/transaction tx: %v", tx)

	resp := &types.MempoolTransactionResponse{
		Transaction: &types.Transaction{
			TransactionIdentifier: &types.TransactionIdentifier{
				Hash: tx.Hash,
			},
		},

		// Transaction: tx,
	}

	return resp, nil
}

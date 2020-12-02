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

package indexer

import (
	"context"
	"errors"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ modules.BalanceStorageHelper = (*BalanceStorageHelper)(nil)

var (
	errNotImplemented = errors.New("not implemented")
)

// BalanceStorageHelper implements storage.BalanceStorageHelper.
type BalanceStorageHelper struct {
	a *asserter.Asserter
}

// AccountBalance attempts to fetch the balance
// for a missing account in storage.
func (h *BalanceStorageHelper) AccountBalance(
	ctx context.Context,
	account *types.AccountIdentifier,
	currency *types.Currency,
	block *types.BlockIdentifier,
) (*types.Amount, error) {
	return &types.Amount{
		Value:    zeroValue,
		Currency: currency,
	}, nil
}

// Asserter returns a *asserter.Asserter.
func (h *BalanceStorageHelper) Asserter() *asserter.Asserter {
	return h.a
}

// BalanceExemptions returns a list of *types.BalanceExemption.
func (h *BalanceStorageHelper) BalanceExemptions() []*types.BalanceExemption {
	return []*types.BalanceExemption{}
}

// ExemptFunc returns a parser.ExemptOperation.
func (h *BalanceStorageHelper) ExemptFunc() parser.ExemptOperation {
	return func(op *types.Operation) bool {
		return false
	}
}

// AccountsReconciled returns the total accounts reconciled by count.
func (h *BalanceStorageHelper) AccountsReconciled(
	ctx context.Context,
	dbTx database.Transaction,
) (*big.Int, error) {
	return nil, errNotImplemented
}

// AccountsSeen returns the total accounts seen by count.
func (h *BalanceStorageHelper) AccountsSeen(
	ctx context.Context,
	dbTx database.Transaction,
) (*big.Int, error) {
	return nil, errNotImplemented
}

package indexer

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/storage"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ storage.BalanceStorageHelper = (*BalanceStorageHelper)(nil)

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
		Value:    "0",
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

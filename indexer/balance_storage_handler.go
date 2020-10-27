package indexer

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/storage"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ storage.BalanceStorageHandler = (*BalanceStorageHandler)(nil)

// BalanceStorageHandler implements storage.BalanceStorageHandler.
type BalanceStorageHandler struct{}

// BlockAdded is called whenever a block is committed to BlockStorage.
func (h *BalanceStorageHandler) BlockAdded(
	ctx context.Context,
	block *types.Block,
	changes []*parser.BalanceChange,
) error {
	return nil
}

// BlockRemoved is called whenever a block is removed from BlockStorage.
func (h *BalanceStorageHandler) BlockRemoved(
	ctx context.Context,
	block *types.Block,
	changes []*parser.BalanceChange,
) error {
	return nil
}

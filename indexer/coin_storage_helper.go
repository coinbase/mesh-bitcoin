package indexer

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/storage"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ storage.CoinStorageHelper = (*CoinStorageHelper)(nil)

type CoinStorageHelper struct {
	b *storage.BlockStorage
}

// CurrentBlockIdentifier returns the current head block identifier
// and is used to comply with the CoinStorageHelper interface.
func (h *CoinStorageHelper) CurrentBlockIdentifier(
	ctx context.Context,
	transaction storage.DatabaseTransaction,
) (*types.BlockIdentifier, error) {
	return h.b.GetHeadBlockIdentifierTransactional(ctx, transaction)
}

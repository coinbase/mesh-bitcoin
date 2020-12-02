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

	"github.com/coinbase/rosetta-sdk-go/storage/database"
	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/coinbase/rosetta-sdk-go/types"
)

var _ modules.CoinStorageHelper = (*CoinStorageHelper)(nil)

// CoinStorageHelper implements storage.CoinStorageHelper.
type CoinStorageHelper struct {
	b *modules.BlockStorage
}

// CurrentBlockIdentifier returns the current head block identifier
// and is used to comply with the CoinStorageHelper interface.
func (h *CoinStorageHelper) CurrentBlockIdentifier(
	ctx context.Context,
	transaction database.Transaction,
) (*types.BlockIdentifier, error) {
	return h.b.GetHeadBlockIdentifierTransactional(ctx, transaction)
}

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
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/coinbase/rosetta-bitcoin/bitcoin"
	"github.com/coinbase/rosetta-bitcoin/configuration"
	mocks "github.com/coinbase/rosetta-bitcoin/mocks/indexer"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func getBlockHash(index int64) string {
	return fmt.Sprintf("block %d", index)
}

var (
	index0 = int64(0)
)

func TestIndexer_Pruning(t *testing.T) {
	// Create Indexer
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)
	defer utils.RemoveTempDir(newDir)

	mockClient := &mocks.Client{}
	pruneDepth := int64(10)
	minHeight := int64(200)
	cfg := &configuration.Configuration{
		Network: &types.NetworkIdentifier{
			Network:    bitcoin.MainnetNetwork,
			Blockchain: bitcoin.Blockchain,
		},
		MaxSyncConcurrency:     256,
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
		Pruning: &configuration.PruningConfiguration{
			Frequency: 50 * time.Millisecond,
			Depth:     pruneDepth,
			MinHeight: minHeight,
		},
		IndexerPath: newDir,
	}

	i, err := Initialize(ctx, cancel, cfg, mockClient)
	assert.NoError(t, err)

	// Waiting for bitcoind...
	mockClient.On("NetworkStatus", ctx).Return(nil, errors.New("not ready")).Once()
	mockClient.On("NetworkStatus", ctx).Return(&types.NetworkStatusResponse{}, nil).Once()

	// Sync to 1000
	mockClient.On("NetworkStatus", ctx).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: 1000,
		},
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
	}, nil)

	// Timeout on first request
	mockClient.On(
		"PruneBlockchain",
		mock.Anything,
		mock.Anything,
	).Return(
		int64(-1),
		errors.New("connection timeout"),
	).Once()

	// Requests after should work
	mockClient.On(
		"PruneBlockchain",
		mock.Anything,
		mock.Anything,
	).Return(
		int64(100),
		nil,
	).Run(
		func(args mock.Arguments) {
			currBlockResponse, err := i.GetBlockLazy(ctx, nil)
			currBlock := currBlockResponse.Block
			assert.NoError(t, err)
			pruningIndex := args.Get(1).(int64)
			assert.True(t, currBlock.BlockIdentifier.Index-pruningIndex >= pruneDepth)
			assert.True(t, pruningIndex >= minHeight)
		},
	)

	// Add blocks
	waitForCheck := make(chan struct{})
	for i := int64(0); i <= 1000; i++ {
		identifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i),
			Index: i,
		}
		parentIdentifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i - 1),
			Index: i - 1,
		}
		if parentIdentifier.Index < 0 {
			parentIdentifier.Index = 0
			parentIdentifier.Hash = getBlockHash(0)
		}

		block := &bitcoin.Block{
			Hash:              identifier.Hash,
			Height:            identifier.Index,
			PreviousBlockHash: parentIdentifier.Hash,
		}
		mockClient.On(
			"GetRawBlock",
			mock.Anything,
			&types.PartialBlockIdentifier{Index: &identifier.Index},
		).Return(
			block,
			[]string{},
			nil,
		).Once()

		blockReturn := &types.Block{
			BlockIdentifier:       identifier,
			ParentBlockIdentifier: parentIdentifier,
			Timestamp:             1599002115110,
		}
		if i != 200 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				map[string]*types.AccountCoin{},
			).Return(
				blockReturn,
				nil,
			).Once()
		} else {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				map[string]*types.AccountCoin{},
			).Return(
				blockReturn,
				nil,
			).Run(func(args mock.Arguments) {
				close(waitForCheck)
			}).Once()
		}
	}

	go func() {
		err := i.Sync(ctx)
		assert.True(t, errors.Is(err, context.Canceled))
	}()

	go func() {
		err := i.Prune(ctx)
		assert.True(t, errors.Is(err, context.Canceled))
	}()

	<-waitForCheck
	waitForFinish := make(chan struct{})
	go func() {
		for {
			currBlockResponse, err := i.GetBlockLazy(ctx, nil)
			if currBlockResponse == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currBlock := currBlockResponse.Block
			assert.NoError(t, err)

			if currBlock.BlockIdentifier.Index == 1000 {
				cancel()
				close(waitForFinish)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	<-waitForFinish
	mockClient.AssertExpectations(t)
}

func TestIndexer_Transactions(t *testing.T) {
	// Create Indexer
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)
	defer utils.RemoveTempDir(newDir)

	mockClient := &mocks.Client{}
	cfg := &configuration.Configuration{
		Network: &types.NetworkIdentifier{
			Network:    bitcoin.MainnetNetwork,
			Blockchain: bitcoin.Blockchain,
		},
		MaxSyncConcurrency:     256,
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
		IndexerPath:            newDir,
	}

	i, err := Initialize(ctx, cancel, cfg, mockClient)
	assert.NoError(t, err)

	// Sync to 1000
	mockClient.On("NetworkStatus", ctx).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: 1000,
		},
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
	}, nil)

	// Add blocks
	waitForCheck := make(chan struct{})
	type coinBankEntry struct {
		Script  *bitcoin.ScriptPubKey
		Coin    *types.Coin
		Account *types.AccountIdentifier
	}

	coinBank := map[string]*coinBankEntry{}
	for i := int64(0); i <= 1000; i++ {
		identifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i),
			Index: i,
		}
		parentIdentifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i - 1),
			Index: i - 1,
		}
		if parentIdentifier.Index < 0 {
			parentIdentifier.Index = 0
			parentIdentifier.Hash = getBlockHash(0)
		}

		transactions := []*types.Transaction{}
		for j := 0; j < 5; j++ {
			rawHash := fmt.Sprintf("block %d transaction %d", i, j)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawHash)))
			coinIdentifier := fmt.Sprintf("%s:%d", hash, index0)
			scriptPubKey := &bitcoin.ScriptPubKey{
				ASM: coinIdentifier,
			}
			marshal, err := types.MarshalMap(scriptPubKey)
			assert.NoError(t, err)
			tx := &types.Transaction{
				TransactionIdentifier: &types.TransactionIdentifier{
					Hash: hash,
				},
				Operations: []*types.Operation{
					{
						OperationIdentifier: &types.OperationIdentifier{
							Index:        0,
							NetworkIndex: &index0,
						},
						Status: types.String(bitcoin.SuccessStatus),
						Type:   bitcoin.OutputOpType,
						Account: &types.AccountIdentifier{
							Address: rawHash,
						},
						Amount: &types.Amount{
							Value:    fmt.Sprintf("%d", rand.Intn(1000)),
							Currency: bitcoin.TestnetCurrency,
						},
						CoinChange: &types.CoinChange{
							CoinAction: types.CoinCreated,
							CoinIdentifier: &types.CoinIdentifier{
								Identifier: coinIdentifier,
							},
						},
						Metadata: map[string]interface{}{
							"scriptPubKey": marshal,
						},
					},
				},
			}
			coinBank[coinIdentifier] = &coinBankEntry{
				Script: scriptPubKey,
				Coin: &types.Coin{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: coinIdentifier,
					},
					Amount: tx.Operations[0].Amount,
				},
				Account: &types.AccountIdentifier{
					Address: rawHash,
				},
			}

			transactions = append(transactions, tx)
		}

		block := &bitcoin.Block{
			Hash:              identifier.Hash,
			Height:            identifier.Index,
			PreviousBlockHash: parentIdentifier.Hash,
		}

		// Require coins that will make the indexer
		// wait.
		requiredCoins := []string{}
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for k := i - 1; k >= 0 && k > i-20; k-- {
			rawHash := fmt.Sprintf("block %d transaction %d", k, rand.Intn(5))
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawHash)))
			requiredCoins = append(requiredCoins, hash+":0")
		}

		mockClient.On(
			"GetRawBlock",
			mock.Anything,
			&types.PartialBlockIdentifier{Index: &identifier.Index},
		).Return(
			block,
			requiredCoins,
			nil,
		).Once()

		blockReturn := &types.Block{
			BlockIdentifier:       identifier,
			ParentBlockIdentifier: parentIdentifier,
			Timestamp:             1599002115110,
			Transactions:          transactions,
		}

		coinMap := map[string]*types.AccountCoin{}
		for _, coinIdentifier := range requiredCoins {
			coinMap[coinIdentifier] = &types.AccountCoin{
				Account: coinBank[coinIdentifier].Account,
				Coin:    coinBank[coinIdentifier].Coin,
			}
		}

		if i != 200 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				coinMap,
			).Return(
				blockReturn,
				nil,
			).Once()
		} else {
			mockClient.On("ParseBlock", mock.Anything, block, coinMap).Return(blockReturn, nil).Run(func(args mock.Arguments) {
				close(waitForCheck)
			}).Once()
		}
	}

	go func() {
		err := i.Sync(ctx)
		assert.True(t, errors.Is(err, context.Canceled))
	}()

	<-waitForCheck
	waitForFinish := make(chan struct{})
	go func() {
		for {
			currBlockResponse, err := i.GetBlockLazy(ctx, nil)
			if currBlockResponse == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currBlock := currBlockResponse.Block
			assert.NoError(t, err)

			if currBlock.BlockIdentifier.Index == 1000 {
				// Ensure ScriptPubKeys are accessible.
				allCoins := []*types.Coin{}
				expectedPubKeys := []*bitcoin.ScriptPubKey{}
				for k, v := range coinBank {
					allCoins = append(allCoins, &types.Coin{
						CoinIdentifier: &types.CoinIdentifier{Identifier: k},
						Amount: &types.Amount{
							Value:    fmt.Sprintf("-%s", v.Coin.Amount.Value),
							Currency: bitcoin.TestnetCurrency,
						},
					})
					expectedPubKeys = append(expectedPubKeys, v.Script)
				}

				pubKeys, err := i.GetScriptPubKeys(ctx, allCoins)
				assert.NoError(t, err)
				assert.Equal(t, expectedPubKeys, pubKeys)

				cancel()
				close(waitForFinish)
				return
			}

			time.Sleep(1 * time.Second)
		}
	}()

	<-waitForFinish
	mockClient.AssertExpectations(t)
}

func TestIndexer_Reorg(t *testing.T) {
	// Create Indexer
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)
	defer utils.RemoveTempDir(newDir)

	mockClient := &mocks.Client{}
	cfg := &configuration.Configuration{
		Network: &types.NetworkIdentifier{
			Network:    bitcoin.MainnetNetwork,
			Blockchain: bitcoin.Blockchain,
		},
		MaxSyncConcurrency:     256,
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
		IndexerPath:            newDir,
	}

	i, err := Initialize(ctx, cancel, cfg, mockClient)
	assert.NoError(t, err)

	// Sync to 1000
	mockClient.On("NetworkStatus", ctx).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: 1000,
		},
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
	}, nil)

	// Add blocks
	waitForCheck := make(chan struct{})
	type coinBankEntry struct {
		Script  *bitcoin.ScriptPubKey
		Coin    *types.Coin
		Account *types.AccountIdentifier
	}

	coinBank := map[string]*coinBankEntry{}

	for i := int64(0); i <= 1000; i++ {
		identifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i),
			Index: i,
		}
		parentIdentifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i - 1),
			Index: i - 1,
		}
		if parentIdentifier.Index < 0 {
			parentIdentifier.Index = 0
			parentIdentifier.Hash = getBlockHash(0)
		}

		transactions := []*types.Transaction{}
		for j := 0; j < 5; j++ {
			rawHash := fmt.Sprintf("block %d transaction %d", i, j)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawHash)))
			coinIdentifier := fmt.Sprintf("%s:%d", hash, index0)
			scriptPubKey := &bitcoin.ScriptPubKey{
				ASM: coinIdentifier,
			}
			marshal, err := types.MarshalMap(scriptPubKey)
			assert.NoError(t, err)
			tx := &types.Transaction{
				TransactionIdentifier: &types.TransactionIdentifier{
					Hash: hash,
				},
				Operations: []*types.Operation{
					{
						OperationIdentifier: &types.OperationIdentifier{
							Index:        0,
							NetworkIndex: &index0,
						},
						Status: types.String(bitcoin.SuccessStatus),
						Type:   bitcoin.OutputOpType,
						Account: &types.AccountIdentifier{
							Address: rawHash,
						},
						Amount: &types.Amount{
							Value:    fmt.Sprintf("%d", rand.Intn(1000)),
							Currency: bitcoin.TestnetCurrency,
						},
						CoinChange: &types.CoinChange{
							CoinAction: types.CoinCreated,
							CoinIdentifier: &types.CoinIdentifier{
								Identifier: coinIdentifier,
							},
						},
						Metadata: map[string]interface{}{
							"scriptPubKey": marshal,
						},
					},
				},
			}
			coinBank[coinIdentifier] = &coinBankEntry{
				Script: scriptPubKey,
				Coin: &types.Coin{
					CoinIdentifier: &types.CoinIdentifier{
						Identifier: coinIdentifier,
					},
					Amount: tx.Operations[0].Amount,
				},
				Account: &types.AccountIdentifier{
					Address: rawHash,
				},
			}
			transactions = append(transactions, tx)
		}

		block := &bitcoin.Block{
			Hash:              identifier.Hash,
			Height:            identifier.Index,
			PreviousBlockHash: parentIdentifier.Hash,
		}

		// Require coins that will make the indexer
		// wait.
		requiredCoins := []string{}
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for k := i - 1; k >= 0 && k > i-20; k-- {
			rawHash := fmt.Sprintf("block %d transaction %d", k, rand.Intn(5))
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawHash)))
			requiredCoins = append(requiredCoins, hash+":0")
		}

		if i == 400 {
			// we will need to call 400 twice
			mockClient.On(
				"GetRawBlock",
				mock.Anything,
				&types.PartialBlockIdentifier{Index: &identifier.Index},
			).Return(
				block,
				requiredCoins,
				nil,
			).Once()
		}

		if i == 401 {
			// require non-existent coins that will never be
			// found to ensure we re-org via abort (with no change
			// in block identifiers)
			mockClient.On(
				"GetRawBlock",
				mock.Anything,
				&types.PartialBlockIdentifier{Index: &identifier.Index},
			).Return(
				block,
				[]string{"blah:1", "blah2:2"},
				nil,
			).Once()
		}

		mockClient.On(
			"GetRawBlock",
			mock.Anything,
			&types.PartialBlockIdentifier{Index: &identifier.Index},
		).Return(
			block,
			requiredCoins,
			nil,
		).Once()

		blockReturn := &types.Block{
			BlockIdentifier:       identifier,
			ParentBlockIdentifier: parentIdentifier,
			Timestamp:             1599002115110,
			Transactions:          transactions,
		}

		coinMap := map[string]*types.AccountCoin{}
		for _, coinIdentifier := range requiredCoins {
			coinMap[coinIdentifier] = &types.AccountCoin{
				Account: coinBank[coinIdentifier].Account,
				Coin:    coinBank[coinIdentifier].Coin,
			}
		}

		if i == 400 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				coinMap,
			).Return(
				blockReturn,
				nil,
			).Once()
		}

		if i != 200 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				coinMap,
			).Return(
				blockReturn,
				nil,
			).Once()
		} else {
			mockClient.On("ParseBlock", mock.Anything, block, coinMap).Return(blockReturn, nil).Run(func(args mock.Arguments) {
				close(waitForCheck)
			}).Once()
		}
	}

	go func() {
		err := i.Sync(ctx)
		assert.True(t, errors.Is(err, context.Canceled))
	}()

	<-waitForCheck
	waitForFinish := make(chan struct{})
	go func() {
		for {
			currBlockResponse, err := i.GetBlockLazy(ctx, nil)
			if currBlockResponse == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currBlock := currBlockResponse.Block
			assert.NoError(t, err)

			if currBlock.BlockIdentifier.Index == 1000 {
				cancel()
				close(waitForFinish)

				return
			}

			time.Sleep(1 * time.Second)
		}
	}()

	<-waitForFinish
	assert.Len(t, i.waiter.table, 0)
	mockClient.AssertExpectations(t)
}

func TestIndexer_HeaderReorg(t *testing.T) {
	// Create Indexer
	ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())

	newDir, err := utils.CreateTempDir()
	assert.NoError(t, err)
	defer utils.RemoveTempDir(newDir)

	mockClient := &mocks.Client{}
	cfg := &configuration.Configuration{
		Network: &types.NetworkIdentifier{
			Network:    bitcoin.MainnetNetwork,
			Blockchain: bitcoin.Blockchain,
		},
		MaxSyncConcurrency:     256,
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
		IndexerPath:            newDir,
	}

	i, err := Initialize(ctx, cancel, cfg, mockClient)
	assert.NoError(t, err)

	// Sync to 1000
	mockClient.On("NetworkStatus", ctx).Return(&types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{
			Index: 1000,
		},
		GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
	}, nil)

	// Add blocks
	waitForCheck := make(chan struct{})
	for i := int64(0); i <= 1000; i++ {
		identifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i),
			Index: i,
		}
		parentIdentifier := &types.BlockIdentifier{
			Hash:  getBlockHash(i - 1),
			Index: i - 1,
		}
		if parentIdentifier.Index < 0 {
			parentIdentifier.Index = 0
			parentIdentifier.Hash = getBlockHash(0)
		}

		transactions := []*types.Transaction{}
		block := &bitcoin.Block{
			Hash:              identifier.Hash,
			Height:            identifier.Index,
			PreviousBlockHash: parentIdentifier.Hash,
		}

		requiredCoins := []string{}
		if i == 400 {
			// we will need to call 400 twice
			mockClient.On(
				"GetRawBlock",
				mock.Anything,
				&types.PartialBlockIdentifier{Index: &identifier.Index},
			).Return(
				block,
				requiredCoins,
				nil,
			).Once()
		}

		if i == 401 {
			// mess up previous block hash to trigger a re-org
			mockClient.On(
				"GetRawBlock",
				mock.Anything,
				&types.PartialBlockIdentifier{Index: &identifier.Index},
			).Return(
				&bitcoin.Block{
					Hash:              identifier.Hash,
					Height:            identifier.Index,
					PreviousBlockHash: "blah",
				},
				[]string{},
				nil,
			).After(5 * time.Second).Once() // we delay to ensure we are at tip here
		}

		mockClient.On(
			"GetRawBlock",
			mock.Anything,
			&types.PartialBlockIdentifier{Index: &identifier.Index},
		).Return(
			block,
			requiredCoins,
			nil,
		).Once()

		blockReturn := &types.Block{
			BlockIdentifier:       identifier,
			ParentBlockIdentifier: parentIdentifier,
			Timestamp:             1599002115110,
			Transactions:          transactions,
		}

		coinMap := map[string]*types.AccountCoin{}
		if i == 400 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				coinMap,
			).Return(
				blockReturn,
				nil,
			).Once()
		}

		if i != 200 {
			mockClient.On(
				"ParseBlock",
				mock.Anything,
				block,
				coinMap,
			).Return(
				blockReturn,
				nil,
			).Once()
		} else {
			mockClient.On("ParseBlock", mock.Anything, block, coinMap).Return(blockReturn, nil).Run(func(args mock.Arguments) {
				close(waitForCheck)
			}).Once()
		}
	}

	go func() {
		err := i.Sync(ctx)
		assert.True(t, errors.Is(err, context.Canceled))
	}()

	<-waitForCheck
	waitForFinish := make(chan struct{})
	go func() {
		for {
			currBlockResponse, err := i.GetBlockLazy(ctx, nil)
			if currBlockResponse == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			currBlock := currBlockResponse.Block
			assert.NoError(t, err)

			if currBlock.BlockIdentifier.Index == 1000 {
				cancel()
				close(waitForFinish)

				return
			}

			time.Sleep(1 * time.Second)
		}
	}()

	<-waitForFinish
	assert.Len(t, i.waiter.table, 0)
	mockClient.AssertExpectations(t)
}

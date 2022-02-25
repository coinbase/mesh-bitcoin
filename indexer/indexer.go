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
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/coinbase/rosetta-bitcoin/bitcoin"
	"github.com/coinbase/rosetta-bitcoin/configuration"
	"github.com/coinbase/rosetta-bitcoin/services"
	"github.com/coinbase/rosetta-bitcoin/utils"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/storage/database"
	storageErrs "github.com/coinbase/rosetta-sdk-go/storage/errors"
	"github.com/coinbase/rosetta-sdk-go/storage/modules"
	"github.com/coinbase/rosetta-sdk-go/syncer"
	"github.com/coinbase/rosetta-sdk-go/types"
	sdkUtils "github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"golang.org/x/sync/semaphore"
)

const (
	// indexPlaceholder is provided to the syncer
	// to indicate we should both start from the
	// last synced block and that we should sync
	// blocks until exit (instead of stopping at
	// a particular height).
	indexPlaceholder = -1

	retryDelay = 10 * time.Second
	retryLimit = 5

	nodeWaitSleep           = 3 * time.Second
	missingTransactionDelay = 200 * time.Millisecond

	// sizeMultiplier is used to multiply the memory
	// estimate for pre-fetching blocks. In other words,
	// this is the estimated memory overhead for each
	// block fetched by the indexer.
	sizeMultiplier = 5

	// zeroValue is 0 as a string
	zeroValue = "0"

	// overclockMultiplier is the amount
	// we multiply runtime.NumCPU by to determine
	// how many goroutines we should
	// spwan to handle block data sequencing.
	overclockMultiplier = 16

	// semaphoreWeight is the weight of each semaphore request.
	semaphoreWeight = int64(1)

	blockLeftShift = 20
)

var (
	errMissingTransaction = errors.New("missing transaction")
)

// Client is used by the indexer to sync blocks.
type Client interface {
	NetworkStatus(context.Context) (*types.NetworkStatusResponse, error)
	PruneBlockchain(context.Context, int64) (int64, error)
	GetRawBlock(context.Context, *types.PartialBlockIdentifier) (*bitcoin.Block, []string, error)
	ParseBlock(
		context.Context,
		*bitcoin.Block,
		map[string]*types.AccountCoin,
	) (*types.Block, error)
}

var _ syncer.Handler = (*Indexer)(nil)
var _ syncer.Helper = (*Indexer)(nil)
var _ services.Indexer = (*Indexer)(nil)

// Indexer caches blocks and provides balance query functionality.
type Indexer struct {
	cancel context.CancelFunc

	network       *types.NetworkIdentifier
	pruningConfig *configuration.PruningConfiguration

	client Client

	asserter       *asserter.Asserter
	database       database.Database
	blockStorage   *modules.BlockStorage
	balanceStorage *modules.BalanceStorage
	coinStorage    *modules.CoinStorage
	workers        []modules.BlockWorker

	waiter *waitTable

	// Store coins created in pre-store before persisted
	// in add block so we can optimistically populate
	// blocks before committed.
	coinCache      map[string]*types.AccountCoin
	coinCacheMutex *sdkUtils.PriorityMutex

	// When populating blocks using pre-stored blocks,
	// we should retry if a new block was seen (similar
	// to trying again if head block changes).
	seen      int64
	seenMutex sync.Mutex

	seenSemaphore *semaphore.Weighted
}

// CloseDatabase closes a storage.Database. This should be called
// before exiting.
func (i *Indexer) CloseDatabase(ctx context.Context) {
	logger := utils.ExtractLogger(ctx, "")
	err := i.database.Close(ctx)
	if err != nil {
		logger.Fatalw("unable to close indexer database", "error", err)
	}

	logger.Infow("database closed successfully")
}

// defaultBadgerOptions returns a set of badger.Options optimized
// for running a Rosetta implementation.
func defaultBadgerOptions(
	dir string,
) badger.Options {
	opts := badger.DefaultOptions(dir)

	// By default, we do not compress the table at all. Doing so can
	// significantly increase memory usage.
	opts.Compression = options.None

	// Load tables into standard I/O value logs.
	opts.TableLoadingMode = options.FileIO
	opts.ValueLogLoadingMode = options.FileIO
	
	// Use an extended table size for larger commits.
	opts.MaxTableSize = database.DefaultMaxTableSize

	// Smaller value log sizes means smaller contiguous memory allocations
	// and less RAM usage on cleanup.
	opts.ValueLogFileSize = database.DefaultLogValueSize

	// To allow writes at a faster speed, we create a new memtable as soon as
	// an existing memtable is filled up. This option determines how many
	// memtables should be kept in memory.
	opts.NumMemtables = 1

	// Don't keep multiple memtables in memory. With larger
	// memtable size, this explodes memory usage.
	opts.NumLevelZeroTables = 1
	opts.NumLevelZeroTablesStall = 2

	// This option will have a significant effect the memory. If the level is kept
	// in-memory, read are faster but the tables will be kept in memory. By default,
	// this is set to false.
	opts.KeepL0InMemory = false

	// We don't compact L0 on close as this can greatly delay shutdown time.
	opts.CompactL0OnClose = false

	// LoadBloomsOnOpen=false will improve the db startup speed. This is also
	// a waste to enable with a limited index cache size (as many of the loaded bloom
	// filters will be immediately discarded from the cache).
	opts.LoadBloomsOnOpen = false

	// BlockSize sets the size of any block in SSTable. SSTable is divided into multiple blocks
	// internally. We set each block to 1MB.
	opts.BlockSize = 1 << blockLeftShift // 1MB

	// NumCompactors sets the number of compaction workers to run concurrently.
	opts.NumCompactors = 2

	return opts
}

// Initialize returns a new Indexer.
func Initialize(
	ctx context.Context,
	cancel context.CancelFunc,
	config *configuration.Configuration,
	client Client,
) (*Indexer, error) {
	localStore, err := database.NewBadgerDatabase(
		ctx,
		config.IndexerPath,
		database.WithCompressorEntries(config.Compressors),
		database.WithCustomSettings(defaultBadgerOptions(
			config.IndexerPath,
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to initialize storage", err)
	}

	blockStorage := modules.NewBlockStorage(localStore, runtime.NumCPU()*overclockMultiplier)
	asserter, err := asserter.NewClientWithOptions(
		config.Network,
		config.GenesisBlockIdentifier,
		bitcoin.OperationTypes,
		bitcoin.OperationStatuses,
		services.Errors,
		nil,
		new(asserter.Validations),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to initialize asserter", err)
	}

	i := &Indexer{
		cancel:         cancel,
		network:        config.Network,
		pruningConfig:  config.Pruning,
		client:         client,
		database:       localStore,
		blockStorage:   blockStorage,
		waiter:         newWaitTable(),
		asserter:       asserter,
		coinCache:      map[string]*types.AccountCoin{},
		coinCacheMutex: new(sdkUtils.PriorityMutex),
		seenSemaphore:  semaphore.NewWeighted(int64(runtime.NumCPU())),
	}

	coinStorage := modules.NewCoinStorage(
		localStore,
		&CoinStorageHelper{blockStorage},
		asserter,
	)
	i.coinStorage = coinStorage

	balanceStorage := modules.NewBalanceStorage(localStore)
	balanceStorage.Initialize(
		&BalanceStorageHelper{asserter},
		&BalanceStorageHandler{},
	)
	i.balanceStorage = balanceStorage

	i.workers = []modules.BlockWorker{coinStorage, balanceStorage}

	return i, nil
}

// waitForNode returns once bitcoind is ready to serve
// block queries.
func (i *Indexer) waitForNode(ctx context.Context) error {
	logger := utils.ExtractLogger(ctx, "indexer")
	for {
		_, err := i.client.NetworkStatus(ctx)
		if err == nil {
			return nil
		}

		logger.Infow("waiting for bitcoind...")
		if err := sdkUtils.ContextSleep(ctx, nodeWaitSleep); err != nil {
			return err
		}
	}
}

// Sync attempts to index Bitcoin blocks using
// the bitcoin.Client until stopped.
func (i *Indexer) Sync(ctx context.Context) error {
	if err := i.waitForNode(ctx); err != nil {
		return fmt.Errorf("%w: failed to wait for node", err)
	}

	i.blockStorage.Initialize(i.workers)

	startIndex := int64(indexPlaceholder)
	head, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
	if err == nil {
		startIndex = head.Index + 1
	}

	// Load in previous blocks into syncer cache to handle reorgs.
	// If previously processed blocks exist in storage, they are fetched.
	// Otherwise, none are provided to the cache (the syncer will not attempt
	// a reorg if the cache is empty).
	pastBlocks := i.blockStorage.CreateBlockCache(ctx, syncer.DefaultPastBlockLimit)

	syncer := syncer.New(
		i.network,
		i,
		i,
		i.cancel,
		syncer.WithCacheSize(syncer.DefaultCacheSize),
		syncer.WithSizeMultiplier(sizeMultiplier),
		syncer.WithPastBlocks(pastBlocks),
	)

	return syncer.Sync(ctx, startIndex, indexPlaceholder)
}

// Prune attempts to prune blocks in bitcoind every
// pruneFrequency.
func (i *Indexer) Prune(ctx context.Context) error {
	logger := utils.ExtractLogger(ctx, "pruner")

	tc := time.NewTicker(i.pruningConfig.Frequency)
	defer tc.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Warnw("exiting pruner")
			return ctx.Err()
		case <-tc.C:
			head, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
			if err != nil {
				continue
			}

			// Must meet pruning conditions in bitcoin core
			// Source:
			// https://github.com/bitcoin/bitcoin/blob/a63a26f042134fa80356860c109edb25ac567552/src/rpc/blockchain.cpp#L953-L960
			pruneHeight := head.Index - i.pruningConfig.Depth
			if pruneHeight <= i.pruningConfig.MinHeight {
				logger.Infow("waiting to prune", "min prune height", i.pruningConfig.MinHeight)
				continue
			}

			logger.Infow("attempting to prune bitcoind", "prune height", pruneHeight)
			prunedHeight, err := i.client.PruneBlockchain(ctx, pruneHeight)
			if err != nil {
				logger.Warnw(
					"unable to prune bitcoind",
					"prune height", pruneHeight,
					"error", err,
				)
			} else {
				logger.Infow("pruned bitcoind", "prune height", prunedHeight)
			}
		}
	}
}

// BlockAdded is called by the syncer when a block is added.
func (i *Indexer) BlockAdded(ctx context.Context, block *types.Block) error {
	logger := utils.ExtractLogger(ctx, "indexer")

	err := i.blockStorage.AddBlock(ctx, block)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to add block to storage %s:%d",
			err,
			block.BlockIdentifier.Hash,
			block.BlockIdentifier.Index,
		)
	}

	ops := 0
	for _, transaction := range block.Transactions {
		ops += len(transaction.Operations)
	}

	// clean cache intermediate
	i.coinCacheMutex.Lock(true)
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			if op.CoinChange == nil {
				continue
			}

			if op.CoinChange.CoinAction != types.CoinCreated {
				continue
			}

			delete(i.coinCache, op.CoinChange.CoinIdentifier.Identifier)
		}
	}
	i.coinCacheMutex.Unlock()

	// Look for all remaining waiting transactions associated
	// with the next block that have not yet been closed. We should
	// abort these waits as they will never be closed by a new transaction.
	i.waiter.Lock()
	for txHash, val := range i.waiter.table {
		if val.earliestBlock == block.BlockIdentifier.Index+1 && !val.channelClosed {
			logger.Debugw(
				"aborting channel",
				"hash", block.BlockIdentifier.Hash,
				"index", block.BlockIdentifier.Index,
				"channel", txHash,
			)
			val.channelClosed = true
			val.aborted = true
			close(val.channel)
		}
	}
	i.waiter.Unlock()

	logger.Debugw(
		"block added",
		"hash", block.BlockIdentifier.Hash,
		"index", block.BlockIdentifier.Index,
		"transactions", len(block.Transactions),
		"ops", ops,
	)

	return nil
}

// BlockSeen is called by the syncer when a block is encountered.
func (i *Indexer) BlockSeen(ctx context.Context, block *types.Block) error {
	if err := i.seenSemaphore.Acquire(ctx, semaphoreWeight); err != nil {
		return err
	}
	defer i.seenSemaphore.Release(semaphoreWeight)

	logger := utils.ExtractLogger(ctx, "indexer")

	// load intermediate
	i.coinCacheMutex.Lock(false)
	for _, tx := range block.Transactions {
		for _, op := range tx.Operations {
			if op.CoinChange == nil {
				continue
			}

			// We only care about newly accessible coins.
			if op.CoinChange.CoinAction != types.CoinCreated {
				continue
			}

			i.coinCache[op.CoinChange.CoinIdentifier.Identifier] = &types.AccountCoin{
				Account: op.Account,
				Coin: &types.Coin{
					CoinIdentifier: op.CoinChange.CoinIdentifier,
					Amount:         op.Amount,
				},
			}
		}
	}
	i.coinCacheMutex.Unlock()

	// Update so that lookers know it exists
	i.seenMutex.Lock()
	i.seen++
	i.seenMutex.Unlock()

	err := i.blockStorage.SeeBlock(ctx, block)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to encounter block to storage %s:%d",
			err,
			block.BlockIdentifier.Hash,
			block.BlockIdentifier.Index,
		)
	}

	ops := 0

	// Close channels of all blocks waiting.
	i.waiter.Lock()
	for _, transaction := range block.Transactions {
		ops += len(transaction.Operations)
		txHash := transaction.TransactionIdentifier.Hash
		val, ok := i.waiter.Get(txHash, false)
		if !ok {
			continue
		}

		if val.channelClosed {
			logger.Debugw(
				"channel already closed",
				"hash", block.BlockIdentifier.Hash,
				"index", block.BlockIdentifier.Index,
				"channel", txHash,
			)
			continue
		}

		// Closing channel will cause all listeners to continue
		val.channelClosed = true
		close(val.channel)
	}
	i.waiter.Unlock()

	logger.Debugw(
		"block seen",
		"hash", block.BlockIdentifier.Hash,
		"index", block.BlockIdentifier.Index,
	)

	return nil
}

// BlockRemoved is called by the syncer when a block is removed.
func (i *Indexer) BlockRemoved(
	ctx context.Context,
	blockIdentifier *types.BlockIdentifier,
) error {
	logger := utils.ExtractLogger(ctx, "indexer")
	logger.Debugw(
		"block removed",
		"hash", blockIdentifier.Hash,
		"index", blockIdentifier.Index,
	)
	err := i.blockStorage.RemoveBlock(ctx, blockIdentifier)
	if err != nil {
		return fmt.Errorf(
			"%w: unable to remove block from storage %s:%d",
			err,
			blockIdentifier.Hash,
			blockIdentifier.Index,
		)
	}

	return nil
}

// NetworkStatus is called by the syncer to get the current
// network status.
func (i *Indexer) NetworkStatus(
	ctx context.Context,
	network *types.NetworkIdentifier,
) (*types.NetworkStatusResponse, error) {
	return i.client.NetworkStatus(ctx)
}

func (i *Indexer) findCoin(
	ctx context.Context,
	btcBlock *bitcoin.Block,
	coinIdentifier string,
) (*types.Coin, *types.AccountIdentifier, error) {
	for ctx.Err() == nil {
		startSeen := i.seen
		databaseTransaction := i.database.ReadTransaction(ctx)
		defer databaseTransaction.Discard(ctx)

		coinHeadBlock, err := i.blockStorage.GetHeadBlockIdentifierTransactional(
			ctx,
			databaseTransaction,
		)
		if errors.Is(err, storageErrs.ErrHeadBlockNotFound) {
			if err := sdkUtils.ContextSleep(ctx, missingTransactionDelay); err != nil {
				return nil, nil, err
			}

			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf(
				"%w: unable to get transactional head block identifier",
				err,
			)
		}

		// Attempt to find coin
		coin, owner, err := i.coinStorage.GetCoinTransactional(
			ctx,
			databaseTransaction,
			&types.CoinIdentifier{
				Identifier: coinIdentifier,
			},
		)
		if err == nil {
			return coin, owner, nil
		}

		if !errors.Is(err, storageErrs.ErrCoinNotFound) {
			return nil, nil, fmt.Errorf("%w: unable to lookup coin %s", err, coinIdentifier)
		}

		// Check seen CoinCache
		i.coinCacheMutex.Lock(false)
		accCoin, ok := i.coinCache[coinIdentifier]
		i.coinCacheMutex.Unlock()
		if ok {
			return accCoin.Coin, accCoin.Account, nil
		}

		// Locking here prevents us from adding sending any done
		// signals while we are determining whether or not to add
		// to the WaitTable.
		i.waiter.Lock()

		// Check to see if head block has increased since
		// we created our databaseTransaction.
		currHeadBlock, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
		if err != nil {
			i.waiter.Unlock()
			return nil, nil, fmt.Errorf("%w: unable to get head block identifier", err)
		}

		// If the block has changed, we try to look up the transaction
		// again.
		if types.Hash(currHeadBlock) != types.Hash(coinHeadBlock) || i.seen != startSeen {
			i.waiter.Unlock()
			continue
		}

		// Put Transaction in WaitTable if doesn't already exist (could be
		// multiple listeners)
		transactionHash := bitcoin.TransactionHash(coinIdentifier)
		val, ok := i.waiter.Get(transactionHash, false)
		if !ok {
			val = &waitTableEntry{
				channel:       make(chan struct{}),
				earliestBlock: btcBlock.Height,
			}
		}
		if val.earliestBlock > btcBlock.Height {
			val.earliestBlock = btcBlock.Height
		}
		val.listeners++
		i.waiter.Set(transactionHash, val, false)
		i.waiter.Unlock()

		return nil, nil, errMissingTransaction
	}

	return nil, nil, ctx.Err()
}

func (i *Indexer) checkHeaderMatch(
	ctx context.Context,
	btcBlock *bitcoin.Block,
) error {
	headBlock, err := i.blockStorage.GetHeadBlockIdentifier(ctx)
	if err != nil && !errors.Is(err, storageErrs.ErrHeadBlockNotFound) {
		return fmt.Errorf("%w: unable to lookup head block", err)
	}

	// If block we are trying to process is next but it is not connected, we
	// should return syncer.ErrOrphanHead to manually trigger a reorg.
	if headBlock != nil &&
		btcBlock.Height == headBlock.Index+1 &&
		btcBlock.PreviousBlockHash != headBlock.Hash {
		return syncer.ErrOrphanHead
	}

	return nil
}

func (i *Indexer) findCoins(
	ctx context.Context,
	btcBlock *bitcoin.Block,
	coins []string,
) (map[string]*types.AccountCoin, error) {
	if err := i.checkHeaderMatch(ctx, btcBlock); err != nil {
		return nil, fmt.Errorf("%w: check header match failed", err)
	}

	coinMap := map[string]*types.AccountCoin{}
	remainingCoins := []string{}
	for _, coinIdentifier := range coins {
		coin, owner, err := i.findCoin(
			ctx,
			btcBlock,
			coinIdentifier,
		)
		if err == nil {
			coinMap[coinIdentifier] = &types.AccountCoin{
				Account: owner,
				Coin:    coin,
			}
			continue
		}

		if errors.Is(err, errMissingTransaction) {
			remainingCoins = append(remainingCoins, coinIdentifier)
			continue
		}

		return nil, fmt.Errorf("%w: unable to find coin %s", err, coinIdentifier)
	}

	if len(remainingCoins) == 0 {
		return coinMap, nil
	}

	// Wait for remaining transactions
	shouldAbort := false
	for _, coinIdentifier := range remainingCoins {
		// Wait on Channel
		txHash := bitcoin.TransactionHash(coinIdentifier)
		entry, ok := i.waiter.Get(txHash, true)
		if !ok {
			return nil, fmt.Errorf("transaction %s not in waiter", txHash)
		}

		select {
		case <-entry.channel:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Delete Transaction from WaitTable if last listener
		i.waiter.Lock()
		val, ok := i.waiter.Get(txHash, false)
		if !ok {
			return nil, fmt.Errorf("transaction %s not in waiter", txHash)
		}

		// Don't exit right away to make sure
		// we remove all closed entries from the
		// waiter.
		if val.aborted {
			shouldAbort = true
		}

		val.listeners--
		if val.listeners == 0 {
			i.waiter.Delete(txHash, false)
		} else {
			i.waiter.Set(txHash, val, false)
		}
		i.waiter.Unlock()
	}

	// Wait to exit until we have decremented our listeners
	if shouldAbort {
		return nil, syncer.ErrOrphanHead
	}

	// In the case of a reorg, we may still not be able to find
	// the transactions. So, we need to repeat this same process
	// recursively until we find the transactions we are looking for.
	foundCoins, err := i.findCoins(ctx, btcBlock, remainingCoins)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to get remaining transactions", err)
	}

	for k, v := range foundCoins {
		coinMap[k] = v
	}

	return coinMap, nil
}

// Block is called by the syncer to fetch a block.
func (i *Indexer) Block(
	ctx context.Context,
	network *types.NetworkIdentifier,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.Block, error) {
	// get raw block
	var btcBlock *bitcoin.Block
	var coins []string
	var err error

	retries := 0
	for ctx.Err() == nil {
		btcBlock, coins, err = i.client.GetRawBlock(ctx, blockIdentifier)
		if err == nil {
			break
		}

		retries++
		if retries > retryLimit {
			return nil, fmt.Errorf("%w: unable to get raw block %+v", err, blockIdentifier)
		}

		if err := sdkUtils.ContextSleep(ctx, retryDelay); err != nil {
			return nil, err
		}
	}

	// determine which coins must be fetched and get from coin storage
	coinMap, err := i.findCoins(ctx, btcBlock, coins)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to find input transactions", err)
	}

	// provide to block parsing
	block, err := i.client.ParseBlock(ctx, btcBlock, coinMap)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to parse block %+v", err, blockIdentifier)
	}

	// ensure block is valid
	if err := i.asserter.Block(block); err != nil {
		return nil, fmt.Errorf("%w: block is not valid %+v", err, blockIdentifier)
	}

	return block, nil
}

// GetScriptPubKeys gets the ScriptPubKey for
// a collection of *types.CoinIdentifier. It also
// confirms that the amount provided with each coin
// is valid.
func (i *Indexer) GetScriptPubKeys(
	ctx context.Context,
	coins []*types.Coin,
) ([]*bitcoin.ScriptPubKey, error) {
	databaseTransaction := i.database.ReadTransaction(ctx)
	defer databaseTransaction.Discard(ctx)

	scripts := make([]*bitcoin.ScriptPubKey, len(coins))
	for j, coin := range coins {
		coinIdentifier := coin.CoinIdentifier
		transactionHash, networkIndex, err := bitcoin.ParseCoinIdentifier(coinIdentifier)
		if err != nil {
			return nil, fmt.Errorf("%w: unable to parse coin identifier", err)
		}

		_, transaction, err := i.blockStorage.FindTransaction(
			ctx,
			&types.TransactionIdentifier{Hash: transactionHash.String()},
			databaseTransaction,
		)
		if err != nil || transaction == nil {
			return nil, fmt.Errorf(
				"%w: unable to find transaction %s",
				err,
				transactionHash.String(),
			)
		}

		for _, op := range transaction.Operations {
			if op.Type != bitcoin.OutputOpType {
				continue
			}

			if *op.OperationIdentifier.NetworkIndex != int64(networkIndex) {
				continue
			}

			var opMetadata bitcoin.OperationMetadata
			if err := types.UnmarshalMap(op.Metadata, &opMetadata); err != nil {
				return nil, fmt.Errorf(
					"%w: unable to unmarshal operation metadata %+v",
					err,
					op.Metadata,
				)
			}

			if types.Hash(op.Amount.Currency) != types.Hash(coin.Amount.Currency) {
				return nil, fmt.Errorf(
					"currency expected %s does not match coin %s",
					types.PrintStruct(coin.Amount.Currency),
					types.PrintStruct(op.Amount.Currency),
				)
			}

			addition, err := types.AddValues(op.Amount.Value, coin.Amount.Value)
			if err != nil {
				return nil, fmt.Errorf("%w: unable to add op amount and coin amount", err)
			}

			if addition != "0" {
				return nil, fmt.Errorf(
					"coin amount does not match expected with difference %s",
					addition,
				)
			}

			scripts[j] = opMetadata.ScriptPubKey
			break
		}

		if scripts[j] == nil {
			return nil, fmt.Errorf("unable to find script for coin %s", coinIdentifier.Identifier)
		}
	}

	return scripts, nil
}

// GetBlockLazy returns a *types.BlockResponse from the indexer's block storage.
// All transactions in a block must be fetched individually.
func (i *Indexer) GetBlockLazy(
	ctx context.Context,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.BlockResponse, error) {
	return i.blockStorage.GetBlockLazy(ctx, blockIdentifier)
}

// GetBlockTransaction returns a *types.Transaction if it is in the provided
// *types.BlockIdentifier.
func (i *Indexer) GetBlockTransaction(
	ctx context.Context,
	blockIdentifier *types.BlockIdentifier,
	transactionIdentifier *types.TransactionIdentifier,
) (*types.Transaction, error) {
	return i.blockStorage.GetBlockTransaction(
		ctx,
		blockIdentifier,
		transactionIdentifier,
	)
}

// GetCoins returns all unspent coins for a particular *types.AccountIdentifier.
func (i *Indexer) GetCoins(
	ctx context.Context,
	accountIdentifier *types.AccountIdentifier,
) ([]*types.Coin, *types.BlockIdentifier, error) {
	return i.coinStorage.GetCoins(ctx, accountIdentifier)
}

// GetBalance returns the balance of an account
// at a particular *types.PartialBlockIdentifier.
func (i *Indexer) GetBalance(
	ctx context.Context,
	accountIdentifier *types.AccountIdentifier,
	currency *types.Currency,
	blockIdentifier *types.PartialBlockIdentifier,
) (*types.Amount, *types.BlockIdentifier, error) {
	dbTx := i.database.ReadTransaction(ctx)
	defer dbTx.Discard(ctx)

	blockResponse, err := i.blockStorage.GetBlockLazyTransactional(
		ctx,
		blockIdentifier,
		dbTx,
	)
	if err != nil {
		return nil, nil, err
	}

	amount, err := i.balanceStorage.GetBalanceTransactional(
		ctx,
		dbTx,
		accountIdentifier,
		currency,
		blockResponse.Block.BlockIdentifier.Index,
	)
	if errors.Is(err, storageErrs.ErrAccountMissing) {
		return &types.Amount{
			Value:    zeroValue,
			Currency: currency,
		}, blockResponse.Block.BlockIdentifier, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return amount, blockResponse.Block.BlockIdentifier, nil
}

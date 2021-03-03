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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coinbase/rosetta-defichain/configuration"
	"github.com/coinbase/rosetta-defichain/defichain"
	"github.com/coinbase/rosetta-defichain/indexer"
	"github.com/coinbase/rosetta-defichain/services"
	"github.com/coinbase/rosetta-defichain/utils"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// readTimeout is the maximum duration for reading the entire
	// request, including the body.
	readTimeout = 5 * time.Second

	// writeTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read.
	writeTimeout = 15 * time.Second

	// idleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	idleTimeout = 30 * time.Second
)

var (
	signalReceived = false
)

// handleSignals handles OS signals so we can ensure we close database
// correctly. We call multiple sigListeners because we
// may need to cancel more than 1 context.
func handleSignals(ctx context.Context, listeners []context.CancelFunc) {
	logger := utils.ExtractLogger(ctx, "signal handler")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Warnw("received signal", "signal", sig)
		signalReceived = true
		for _, listener := range listeners {
			listener()
		}
	}()
}

func startOnlineDependencies(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configuration.Configuration,
	g *errgroup.Group,
) (*defichain.Client, *indexer.Indexer, error) {
	client := defichain.NewClient(
		defichain.LocalhostURL(cfg.RPCPort),
		cfg.GenesisBlockIdentifier,
		cfg.Currency,
	)

	g.Go(func() error {
		return defichain.StartDefichaind(ctx, cfg.ConfigPath, g)
	})

	i, err := indexer.Initialize(
		ctx,
		cancel,
		cfg,
		client,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: unable to initialize indexer", err)
	}

	g.Go(func() error {
		err = i.Sync(ctx)
		if err != nil {
			fmt.Printf("sync error: %v", err)
		}
		return err
	})

	g.Go(func() error {
		err = i.Prune(ctx)
		if err != nil {
			fmt.Printf("prune error: %v", err)
		}
		return err
	})

	return client, i, nil
}

func main() {
	loggerRaw, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	defer func() {
		_ = loggerRaw.Sync()
	}()

	ctx := context.Background()
	ctx = ctxzap.ToContext(ctx, loggerRaw)
	ctx, cancel := context.WithCancel(ctx)
	go handleSignals(ctx, []context.CancelFunc{cancel})

	logger := loggerRaw.Sugar().Named("main")

	cfg, err := configuration.LoadConfiguration(configuration.DataDirectory)
	if err != nil {
		logger.Fatalw("unable to load configuration", "error", err)
	}

	logger.Infow("loaded configuration", "configuration", types.PrintStruct(cfg))

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return utils.MonitorMemoryUsage(ctx, -1)
	})

	var i *indexer.Indexer
	var client *defichain.Client
	if cfg.Mode == configuration.Online {
		client, i, err = startOnlineDependencies(ctx, cancel, cfg, g)
		if err != nil {
			logger.Fatalw("unable to start online dependencies", "error", err)
		}
	}

	// The asserter automatically rejects incorrectly formatted
	// requests.
	asserter, err := asserter.NewServer(
		defichain.OperationTypes,
		services.HistoricalBalanceLookup,
		[]*types.NetworkIdentifier{cfg.Network},
		nil,
		services.MempoolCoins,
	)
	if err != nil {
		logger.Fatalw("unable to create new server asserter", "error", err)
	}

	router := services.NewBlockchainRouter(cfg, client, i, asserter)
	loggedRouter := services.LoggerMiddleware(loggerRaw, router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      corsRouter,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	g.Go(func() error {
		logger.Infow("server listening", "port", cfg.Port)
		return server.ListenAndServe()
	})

	g.Go(func() error {
		// If we don't shutdown server in errgroup, it will
		// never stop because server.ListenAndServe doesn't
		// take any context.
		<-ctx.Done()

		return server.Shutdown(ctx)
	})

	err = g.Wait()

	// We always want to attempt to close the database, regardless of the error.
	// We also want to do this after all indexer goroutines have stopped.
	if i != nil {
		i.CloseDatabase(ctx)
	}

	if signalReceived {
		logger.Fatalw("rosetta-bitcoin halted")
	}

	if err != nil {
		logger.Fatalw("rosetta-bitcoin sync failed", "error", err)
	}
}

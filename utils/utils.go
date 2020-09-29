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

package utils

import (
	"context"
	"time"

	sdkUtils "github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	// monitorMemorySleep is how long we should sleep
	// between checking memory stats.
	monitorMemorySleep = 50 * time.Millisecond
)

// ExtractLogger returns a sugared logger with the origin
// tag added.
func ExtractLogger(ctx context.Context, origin string) *zap.SugaredLogger {
	logger := ctxzap.Extract(ctx)
	if len(origin) > 0 {
		logger = logger.Named(origin)
	}

	return logger.Sugar()
}

// MonitorMemoryUsage periodically logs memory usage
// stats and triggers garbage collection when heap allocations
// surpass maxHeapUsage.
func MonitorMemoryUsage(
	ctx context.Context,
	maxHeapUsage int,
) error {
	logger := ExtractLogger(ctx, "memory")

	maxHeap := float64(0)
	garbageCollections := uint32(0)
	for ctx.Err() == nil {
		memUsage := sdkUtils.MonitorMemoryUsage(ctx, maxHeapUsage)
		if memUsage.Heap > maxHeap {
			maxHeap = memUsage.Heap
		}

		if memUsage.GarbageCollections > garbageCollections {
			garbageCollections = memUsage.GarbageCollections
			logger.Debugw(
				"stats",
				"heap (MB)", memUsage.Heap,
				"max heap (MB)", maxHeap,
				"stack (MB)", memUsage.Stack,
				"system (MB)", memUsage.System,
				"garbage collections", memUsage.GarbageCollections,
			)
		}

		if err := sdkUtils.ContextSleep(ctx, monitorMemorySleep); err != nil {
			return err
		}
	}

	return ctx.Err()
}

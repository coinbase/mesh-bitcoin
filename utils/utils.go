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
	"runtime"
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

// ContextSleep sleeps for the provided duration and returns
// an error if context is canceled.
func ContextSleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			return nil
		}
	}
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
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		heapAlloc := float64(m.HeapAlloc)
		if heapAlloc > maxHeap {
			maxHeap = heapAlloc
		}

		heapAllocMB := sdkUtils.BtoMb(heapAlloc)
		if heapAllocMB > float64(maxHeapUsage) {
			runtime.GC()
		}

		if m.NumGC > garbageCollections {
			garbageCollections = m.NumGC
			logger.Debugw(
				"stats",
				"heap (MB)", heapAllocMB,
				"max heap (MB)", sdkUtils.BtoMb(maxHeap),
				"stack (MB)", sdkUtils.BtoMb(float64(m.StackInuse)),
				"system (MB)", sdkUtils.BtoMb(float64(m.Sys)),
				"garbage collections", m.NumGC,
			)
		}

		if err := ContextSleep(ctx, monitorMemorySleep); err != nil {
			return err
		}
	}

	return ctx.Err()
}

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
	"sync"
)

type waitTable struct {
	table map[string]*waitTableEntry

	// lock is held when we want to perform multiple
	// reads/writes to waiter.
	lock sync.Mutex
}

func newWaitTable() *waitTable {
	return &waitTable{
		table: map[string]*waitTableEntry{},
	}
}

func (t *waitTable) Lock() {
	t.lock.Lock()
}

func (t *waitTable) Unlock() {
	t.lock.Unlock()
}

func (t *waitTable) Get(key string, safe bool) (*waitTableEntry, bool) {
	if safe {
		t.lock.Lock()
		defer t.lock.Unlock()
	}

	v, ok := t.table[key]
	if !ok {
		return nil, false
	}

	return v, true
}

func (t *waitTable) Set(key string, value *waitTableEntry, safe bool) {
	if safe {
		t.lock.Lock()
		defer t.lock.Unlock()
	}
	t.table[key] = value
}

func (t *waitTable) Delete(key string, safe bool) {
	if safe {
		t.lock.Lock()
		defer t.lock.Unlock()
	}

	delete(t.table, key)
}

type waitTableEntry struct {
	listeners int // need to know when to delete entry (i.e. when no listeners)
	channel   chan struct{}

	channelClosed bool // needed to know if we should abort (can't read if channel is closed)
	aborted       bool
	earliestBlock int64
}

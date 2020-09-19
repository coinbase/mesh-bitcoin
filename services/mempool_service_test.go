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

package services

import (
	"context"
	"testing"

	mocks "github.com/coinbase/rosetta-bitcoin/mocks/services"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestMempoolEndpoints(t *testing.T) {
	mockClient := &mocks.Client{}
	servicer := NewMempoolAPIService(mockClient)
	ctx := context.Background()

	mockClient.On("RawMempool", ctx).Return([]string{
		"tx1",
		"tx2",
	}, nil)
	mem, err := servicer.Mempool(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, &types.MempoolResponse{
		TransactionIdentifiers: []*types.TransactionIdentifier{
			{
				Hash: "tx1",
			},
			{
				Hash: "tx2",
			},
		},
	}, mem)

	memTransaction, err := servicer.MempoolTransaction(ctx, nil)
	assert.Nil(t, memTransaction)
	assert.Equal(t, ErrUnimplemented.Code, err.Code)
	assert.Equal(t, ErrUnimplemented.Message, err.Message)
	mockClient.AssertExpectations(t)
}

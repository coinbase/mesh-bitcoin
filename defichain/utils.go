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

package defichain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// ParseCoinIdentifier returns the corresponding hash and index associated
// with a *types.CoinIdentifier.
func ParseCoinIdentifier(coinIdentifier *types.CoinIdentifier) (*chainhash.Hash, uint32, error) {
	utxoSpent := strings.Split(coinIdentifier.Identifier, ":")

	outpointHash := utxoSpent[0]
	if len(outpointHash) != TransactionHashLength {
		return nil, 0, fmt.Errorf("outpoint_hash %s is not length 64", outpointHash)
	}

	hash, err := chainhash.NewHashFromStr(outpointHash)
	if err != nil {
		return nil, 0, fmt.Errorf("%w unable to construct has from string %s", err, outpointHash)
	}

	outpointIndex, err := strconv.ParseUint(utxoSpent[1], 10, 32)
	if err != nil {
		return nil, 0, fmt.Errorf("%w unable to parse outpoint_index", err)
	}

	return hash, uint32(outpointIndex), nil
}

// ParseSingleAddress extracts a single address from a pkscript or
// throws an error.
func ParseSingleAddress(
	chainParams *chaincfg.Params,
	script []byte,
) (txscript.ScriptClass, btcutil.Address, error) {
	class, addresses, nRequired, err := txscript.ExtractPkScriptAddrs(script, chainParams)
	if err != nil {
		return 0, nil, fmt.Errorf("%w unable to extract script addresses", err)
	}

	if nRequired != 1 {
		return 0, nil, fmt.Errorf("expecting 1 address, got %d", nRequired)
	}

	address := addresses[0]

	return class, address, nil
}

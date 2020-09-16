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

package configuration

import (
	"errors"
	"os"
	"path"
	"testing"

	"github.com/coinbase/rosetta-bitcoin/bitcoin"

	"github.com/coinbase/rosetta-sdk-go/storage"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfiguration(t *testing.T) {
	tests := map[string]struct {
		Mode    string
		Network string
		Port    string

		cfg *Configuration
		err error
	}{
		"no envs set": {
			err: errors.New("MODE must be populated"),
		},
		"only mode set": {
			Mode: string(Online),
			err:  errors.New("NETWORK must be populated"),
		},
		"only mode and network set": {
			Mode:    string(Online),
			Network: Mainnet,
			err:     errors.New("PORT must be populated"),
		},
		"all set (mainnet)": {
			Mode:    string(Online),
			Network: Mainnet,
			Port:    "1000",
			cfg: &Configuration{
				Mode: Online,
				Network: &types.NetworkIdentifier{
					Network:    bitcoin.MainnetNetwork,
					Blockchain: bitcoin.Blockchain,
				},
				Params:                 bitcoin.MainnetParams,
				Currency:               bitcoin.MainnetCurrency,
				GenesisBlockIdentifier: bitcoin.MainnetGenesisBlockIdentifier,
				Port:                   1000,
				RPCPort:                mainnetRPCPort,
				ConfigPath:             mainnetConfigPath,
				Pruning: &PruningConfiguration{
					Frequency: pruneFrequency,
					Depth:     pruneDepth,
					MinHeight: minPruneHeight,
				},
				Compressors: []*storage.CompressorEntry{
					{
						Namespace:      transactionNamespace,
						DictionaryPath: mainnetTransactionDictionary,
					},
				},
			},
		},
		"all set (testnet)": {
			Mode:    string(Online),
			Network: Testnet,
			Port:    "1000",
			cfg: &Configuration{
				Mode: Online,
				Network: &types.NetworkIdentifier{
					Network:    bitcoin.TestnetNetwork,
					Blockchain: bitcoin.Blockchain,
				},
				Params:                 bitcoin.TestnetParams,
				Currency:               bitcoin.TestnetCurrency,
				GenesisBlockIdentifier: bitcoin.TestnetGenesisBlockIdentifier,
				Port:                   1000,
				RPCPort:                testnetRPCPort,
				ConfigPath:             testnetConfigPath,
				Pruning: &PruningConfiguration{
					Frequency: pruneFrequency,
					Depth:     pruneDepth,
					MinHeight: minPruneHeight,
				},
				Compressors: []*storage.CompressorEntry{
					{
						Namespace:      transactionNamespace,
						DictionaryPath: testnetTransactionDictionary,
					},
				},
			},
		},
		"invalid mode": {
			Mode:    "bad mode",
			Network: Testnet,
			Port:    "1000",
			err:     errors.New("bad mode is not a valid mode"),
		},
		"invalid network": {
			Mode:    string(Offline),
			Network: "bad network",
			Port:    "1000",
			err:     errors.New("bad network is not a valid network"),
		},
		"invalid port": {
			Mode:    string(Offline),
			Network: Testnet,
			Port:    "bad port",
			err:     errors.New("unable to parse port bad port"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			newDir, err := utils.CreateTempDir()
			assert.NoError(t, err)
			defer utils.RemoveTempDir(newDir)

			os.Setenv(ModeEnv, test.Mode)
			os.Setenv(NetworkEnv, test.Network)
			os.Setenv(PortEnv, test.Port)

			cfg, err := LoadConfiguration(newDir)
			if test.err != nil {
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), test.err.Error())
			} else {
				test.cfg.IndexerPath = path.Join(newDir, "indexer")
				test.cfg.BitcoindPath = path.Join(newDir, "bitcoind")
				assert.Equal(t, test.cfg, cfg)
				assert.NoError(t, err)
			}
		})
	}
}

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

package bitcoin

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

const (
	url = "/"
)

func forceMarshalMap(t *testing.T, i interface{}) map[string]interface{} {
	m, err := types.MarshalMap(i)
	if err != nil {
		t.Fatalf("could not marshal map %s", types.PrintStruct(i))
	}

	return m
}

var (
	blockIdentifier1000 = &types.BlockIdentifier{
		Hash:  "00000000c937983704a73af28acdec37b049d214adbda81d7e2a3dd146f6ed09",
		Index: 1000,
	}

	block1000 = &Block{
		Hash:              "00000000c937983704a73af28acdec37b049d214adbda81d7e2a3dd146f6ed09",
		Height:            1000,
		PreviousBlockHash: "0000000008e647742775a230787d66fdf92c46a48c896bfbc85cdc8acc67e87d",
		Time:              1232346882,
		Size:              216,
		Weight:            864,
		Version:           1,
		MerkleRoot:        "fe28050b93faea61fa88c4c630f0e1f0a1c24d0082dd0e10d369e13212128f33",
		MedianTime:        1232344831,
		Nonce:             2595206198,
		Bits:              "1d00ffff",
		Difficulty:        1,
		Txs: []*Transaction{
			{
				Hex:      "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0804ffff001d02fd04ffffffff0100f2052a01000000434104f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446aac00000000", // nolint
				Hash:     "fe28050b93faea61fa88c4c630f0e1f0a1c24d0082dd0e10d369e13212128f33",
				Size:     135,
				Vsize:    135,
				Version:  1,
				Locktime: 0,
				Weight:   540,
				Inputs: []*Input{
					{
						Coinbase: "04ffff001d02fd04",
						Sequence: 4294967295,
					},
				},
				Outputs: []*Output{
					{
						Value: 50,
						Index: 0,
						ScriptPubKey: &ScriptPubKey{
							ASM:  "04f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446a OP_CHECKSIG", // nolint
							Hex:  "4104f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446aac",         // nolint
							Type: "pubkey",
						},
					},
				},
			},
			{
				Hex:      "01000000081cefd96060ecb1c4fbe675ad8a4f8bdc61d634c52b3a1c4116dee23749fe80ff000000009300493046022100866859c21f306538152e83f115bcfbf59ab4bb34887a88c03483a5dff9895f96022100a6dfd83caa609bf0516debc2bf65c3df91813a4842650a1858b3f61cfa8af249014730440220296d4b818bb037d0f83f9f7111665f49532dfdcbec1e6b784526e9ac4046eaa602204acf3a5cb2695e8404d80bf49ab04828bcbe6fc31d25a2844ced7a8d24afbdff01ffffffff1cefd96060ecb1c4fbe675ad8a4f8bdc61d634c52b3a1c4116dee23749fe80ff020000009400483045022100e87899175991aa008176cb553c6f2badbb5b741f328c9845fcab89f8b18cae2302200acce689896dc82933015e7230e5230d5cff8a1ffe82d334d60162ac2c5b0c9601493046022100994ad29d1e7b03e41731a4316e5f4992f0d9b6e2efc40a1ccd2c949b461175c502210099b69fdc2db00fbba214f16e286f6a49e2d8a0d5ffc6409d87796add475478d601ffffffff1e4a6d2d280ea06680d6cf8788ac90344a9c67cca9b06005bbd6d3f6945c8272010000009500493046022100a27400ba52fd842ce07398a1de102f710a10c5599545e6c95798934352c2e4df022100f6383b0b14c9f64b6718139f55b6b9494374755b86bae7d63f5d3e583b57255a01493046022100fdf543292f34e1eeb1703b264965339ec4a450ec47585009c606b3edbc5b617b022100a5fbb1c8de8aaaa582988cdb23622838e38de90bebcaab3928d949aa502a65d401ffffffff1e4a6d2d280ea06680d6cf8788ac90344a9c67cca9b06005bbd6d3f6945c8272020000009400493046022100ac626ac3051f875145b4fe4cfe089ea895aac73f65ab837b1ac30f5d875874fa022100bc03e79fa4b7eb707fb735b95ff6613ca33adeaf3a0607cdcead4cfd3b51729801483045022100b720b04a5c5e2f61b7df0fcf334ab6fea167b7aaede5695d3f7c6973496adbf1022043328c4cc1cdc3e5db7bb895ccc37133e960b2fd3ece98350f774596badb387201ffffffff23a8733e349c97d6cd90f520fdd084ba15ce0a395aad03cd51370602bb9e5db3010000004a00483045022100e8556b72c5e9c0da7371913a45861a61c5df434dfd962de7b23848e1a28c86ca02205d41ceda00136267281be0974be132ac4cda1459fe2090ce455619d8b91045e901ffffffff6856d609b881e875a5ee141c235e2a82f6b039f2b9babe82333677a5570285a6000000006a473044022040a1c631554b8b210fbdf2a73f191b2851afb51d5171fb53502a3a040a38d2c0022040d11cf6e7b41fe1b66c3d08f6ada1aee07a047cb77f242b8ecc63812c832c9a012102bcfad931b502761e452962a5976c79158a0f6d307ad31b739611dac6a297c256ffffffff6856d609b881e875a5ee141c235e2a82f6b039f2b9babe82333677a5570285a601000000930048304502205b109df098f7e932fbf71a45869c3f80323974a826ee2770789eae178a21bfc8022100c0e75615e53ee4b6e32b9bb5faa36ac539e9c05fa2ae6b6de5d09c08455c8b9601483045022009fb7d27375c47bea23b24818634df6a54ecf72d52e0c1268fb2a2c84f1885de022100e0ed4f15d62e7f537da0d0f1863498f9c7c0c0a4e00e4679588c8d1a9eb20bb801ffffffffa563c3722b7b39481836d5edfc1461f97335d5d1e9a23ade13680d0e2c1c371f030000006c493046022100ecc38ae2b1565643dc3c0dad5e961a5f0ea09cab28d024f92fa05c922924157e022100ebc166edf6fbe4004c72bfe8cf40130263f98ddff728c8e67b113dbd621906a601210211a4ed241174708c07206601b44a4c1c29e5ad8b1f731c50ca7e1d4b2a06dc1fffffffff02d0223a00000000001976a91445db0b779c0b9fa207f12a8218c94fc77aff504588ac80f0fa02000000000000000000", // nolint
				Hash:     "4852fe372ff7534c16713b3146bbc1e86379c70bea4d5c02fb1fa0112980a081",
				Size:     1408,
				Vsize:    1408,
				Version:  1,
				Locktime: 0,
				Weight:   5632,
				Inputs:   []*Input{}, // all we care about in this test is the outputs
				Outputs: []*Output{
					{
						Value: 0.0381,
						Index: 0,
						ScriptPubKey: &ScriptPubKey{
							ASM:          "OP_DUP OP_HASH160 45db0b779c0b9fa207f12a8218c94fc77aff5045 OP_EQUALVERIFY OP_CHECKSIG",
							Hex:          "76a91445db0b779c0b9fa207f12a8218c94fc77aff504588ac",
							RequiredSigs: 1,
							Type:         "pubkeyhash",
							Addresses: []string{
								"mmtKKnjqTPdkBnBMbNt5Yu2SCwpMaEshEL",
							},
						},
					},
					{
						Value: 0.5,
						Index: 1,
						ScriptPubKey: &ScriptPubKey{
							ASM:  "",
							Hex:  "",
							Type: "nonstandard",
						},
					},
				},
			},
		},
	}

	blockIdentifier100000 = &types.BlockIdentifier{
		Hash:  "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
		Index: 100000,
	}

	block100000 = &Block{
		Hash:              "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
		Height:            100000,
		PreviousBlockHash: "000000000002d01c1fccc21636b607dfd930d31d01c3a62104612a1719011250",
		Time:              1293623863,
		Size:              957,
		Weight:            3828,
		Version:           1,
		MerkleRoot:        "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
		MedianTime:        1293622620,
		Nonce:             274148111,
		Bits:              "1b04864c",
		Difficulty:        14484.1623612254,
		Txs: []*Transaction{
			{
				Hex:      "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff08044c86041b020602ffffffff0100f2052a010000004341041b0e8c2567c12536aa13357b79a073dc4444acb83c4ec7a0e2f99dd7457516c5817242da796924ca4e99947d087fedf9ce467cb9f7c6287078f801df276fdf84ac00000000", // nolint
				Hash:     "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
				Size:     135,
				Vsize:    135,
				Version:  1,
				Locktime: 0,
				Weight:   540,
				Inputs: []*Input{
					{
						Coinbase: "044c86041b020602",
						Sequence: 4294967295,
					},
				},
				Outputs: []*Output{
					{
						Value: 15.89351625,
						Index: 0,
						ScriptPubKey: &ScriptPubKey{
							ASM:          "OP_HASH160 228f554bbf766d6f9cc828de1126e3d35d15e5fe OP_EQUAL",
							Hex:          "a914228f554bbf766d6f9cc828de1126e3d35d15e5fe87",
							RequiredSigs: 1,
							Type:         "scripthash",
							Addresses: []string{
								"34qkc2iac6RsyxZVfyE2S5U5WcRsbg2dpK",
							},
						},
					},
					{
						Value: 0,
						Index: 1,
						ScriptPubKey: &ScriptPubKey{
							ASM:  "OP_RETURN aa21a9ed10109f4b82aa3ed7ec9d02a2a90246478b3308c8b85daf62fe501d58d05727a4",
							Hex:  "6a24aa21a9ed10109f4b82aa3ed7ec9d02a2a90246478b3308c8b85daf62fe501d58d05727a4",
							Type: "nulldata",
						},
					},
				},
			},
			{
				Hex:      "0100000001032e38e9c0a84c6046d687d10556dcacc41d275ec55fc00779ac88fdf357a187000000008c493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3ffffffff0200e32321000000001976a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac000fe208010000001976a914948c765a6914d43f2a7ac177da2c2f6b52de3d7c88ac00000000", // nolint
				Hash:     "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
				Size:     259,
				Vsize:    259,
				Version:  1,
				Locktime: 0,
				Weight:   1036,
				Inputs: []*Input{
					{
						TxHash: "87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03",
						Vout:   0,
						ScriptSig: &ScriptSig{
							ASM: "3046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748[ALL] 04f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
							Hex: "493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
						},
						Sequence: 4294967295,
					},
				},
				Outputs: []*Output{
					{
						Value: 5.56,
						Index: 0,
						ScriptPubKey: &ScriptPubKey{
							ASM:          "OP_DUP OP_HASH160 c398efa9c392ba6013c5e04ee729755ef7f58b32 OP_EQUALVERIFY OP_CHECKSIG",
							Hex:          "76a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac",
							RequiredSigs: 1,
							Type:         "pubkeyhash",
							Addresses: []string{
								"1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
							},
						},
					},
					{
						Value: 44.44,
						Index: 1,
						ScriptPubKey: &ScriptPubKey{
							ASM:          "OP_DUP OP_HASH160 948c765a6914d43f2a7ac177da2c2f6b52de3d7c OP_EQUALVERIFY OP_CHECKSIG",
							Hex:          "76a914948c765a6914d43f2a7ac177da2c2f6b52de3d7c88ac",
							RequiredSigs: 1,
							Type:         "pubkeyhash",
							Addresses: []string{
								"1EYTGtG4LnFfiMvjJdsU7GMGCQvsRSjYhx",
							},
						},
					},
				},
			},
			{
				Hash:     "fake",
				Hex:      "fake hex",
				Version:  2,
				Size:     421,
				Vsize:    612,
				Weight:   129992,
				Locktime: 10,
				Inputs: []*Input{
					{
						TxHash: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5",
						Vout:   0,
						ScriptSig: &ScriptSig{
							ASM: "00142b2296c588ec413cebd19c3cbc04ea830ead6e78",
							Hex: "1600142b2296c588ec413cebd19c3cbc04ea830ead6e78",
						},
						TxInWitness: []string{
							"304402205f39ccbab38b644acea0776d18cb63ce3e37428cbac06dc23b59c61607aef69102206b8610827e9cb853ea0ba38983662034bd3575cc1ab118fb66d6a98066fa0bed01", // nolint
							"0304c01563d46e38264283b99bb352b46e69bf132431f102d4bd9a9d8dab075e7f",
						},
						Sequence: 4294967295,
					},
					{
						TxHash: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5",
						Vout:   1,
						ScriptSig: &ScriptSig{
							ASM: "00142b2296c588ec413cebd19c3cbc04ea830ead6e78",
							Hex: "1600142b2296c588ec413cebd19c3cbc04ea830ead6e78",
						},
						Sequence: 4294967295,
					},
					{
						TxHash: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
						Vout:   0,
						ScriptSig: &ScriptSig{
							ASM: "3046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748[ALL] 04f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
							Hex: "493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
						},
						Sequence: 4294967295,
					},
				},
				Outputs: []*Output{
					{
						Value: 200.56,
						Index: 0,
						ScriptPubKey: &ScriptPubKey{
							ASM:          "OP_DUP OP_HASH160 c398efa9c392ba6013c5e04ee729755ef7f58b32 OP_EQUALVERIFY OP_CHECKSIG",
							Hex:          "76a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac",
							RequiredSigs: 1,
							Type:         "pubkeyhash",
							Addresses: []string{
								"1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
								"1EYTGtG4LnFfiMvjJdsU7GMGCQvsRSjYhx",
							},
						},
					},
				},
			},
		},
	}
)

func TestNetworkStatus(t *testing.T) {
	tests := map[string]struct {
		responses []responseFixture

		expectedStatus *types.NetworkStatusResponse
		expectedError  error
	}{
		"successful": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_blockchain_info_response.json"),
					url:    url,
				},
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response.json"),
					url:    url,
				},
				{
					status: http.StatusOK,
					body:   loadFixture("get_peer_info_response.json"),
					url:    url,
				},
			},
			expectedStatus: &types.NetworkStatusResponse{
				CurrentBlockIdentifier: blockIdentifier1000,
				CurrentBlockTimestamp:  block1000.Time * 1000,
				GenesisBlockIdentifier: MainnetGenesisBlockIdentifier,
				Peers: []*types.Peer{
					{
						PeerID: "77.93.223.9:8333",
						Metadata: forceMarshalMap(t, &PeerInfo{
							Addr:           "77.93.223.9:8333",
							Version:        70015,
							SubVer:         "/Satoshi:0.14.2/",
							StartingHeight: 643579,
							RelayTxes:      true,
							LastSend:       1597606676,
							LastRecv:       1597606677,
							BanScore:       0,
							SyncedHeaders:  644046,
							SyncedBlocks:   644046,
						}),
					},
					{
						PeerID: "172.105.93.179:8333",
						Metadata: forceMarshalMap(t, &PeerInfo{
							Addr:           "172.105.93.179:8333",
							RelayTxes:      true,
							LastSend:       1597606678,
							LastRecv:       1597606676,
							Version:        70015,
							SubVer:         "/Satoshi:0.18.1/",
							StartingHeight: 643579,
							BanScore:       0,
							SyncedHeaders:  644046,
							SyncedBlocks:   644046,
						}),
					},
				},
			},
		},
		"blockchain warming up error": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("rpc_in_warmup_response.json"),
					url:    url,
				},
			},
			expectedError: errors.New("rpc in warmup"),
		},
		"blockchain info error": {
			responses: []responseFixture{
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
		"peer info not accessible": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_blockchain_info_response.json"),
					url:    url,
				},
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response.json"),
					url:    url,
				},
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			responses := make(chan responseFixture, len(test.responses))
			for _, response := range test.responses {
				responses <- response
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := <-responses
				assert.Equal("application/json", r.Header.Get("Content-Type"))
				assert.Equal("POST", r.Method)
				assert.Equal(response.url, r.URL.RequestURI())

				w.WriteHeader(response.status)
				fmt.Fprintln(w, response.body)
			}))

			client := NewClient(ts.URL, MainnetGenesisBlockIdentifier, MainnetCurrency)
			status, err := client.NetworkStatus(context.Background())
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedStatus, status)
			}
		})
	}
}

func TestGetPeers(t *testing.T) {
	tests := map[string]struct {
		responses []responseFixture

		expectedPeers []*types.Peer
		expectedError error
	}{
		"successful": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_peer_info_response.json"),
					url:    url,
				},
			},
			expectedPeers: []*types.Peer{
				{
					PeerID: "77.93.223.9:8333",
					Metadata: forceMarshalMap(t, &PeerInfo{
						Addr:           "77.93.223.9:8333",
						Version:        70015,
						SubVer:         "/Satoshi:0.14.2/",
						StartingHeight: 643579,
						RelayTxes:      true,
						LastSend:       1597606676,
						LastRecv:       1597606677,
						BanScore:       0,
						SyncedHeaders:  644046,
						SyncedBlocks:   644046,
					}),
				},
				{
					PeerID: "172.105.93.179:8333",
					Metadata: forceMarshalMap(t, &PeerInfo{
						Addr:           "172.105.93.179:8333",
						RelayTxes:      true,
						LastSend:       1597606678,
						LastRecv:       1597606676,
						Version:        70015,
						SubVer:         "/Satoshi:0.18.1/",
						StartingHeight: 643579,
						BanScore:       0,
						SyncedHeaders:  644046,
						SyncedBlocks:   644046,
					}),
				},
			},
		},
		"blockchain warming up error": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("rpc_in_warmup_response.json"),
					url:    url,
				},
			},
			expectedError: errors.New("rpc in warmup"),
		},
		"peer info error": {
			responses: []responseFixture{
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			responses := make(chan responseFixture, len(test.responses))
			for _, response := range test.responses {
				responses <- response
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := <-responses
				assert.Equal("application/json", r.Header.Get("Content-Type"))
				assert.Equal("POST", r.Method)
				assert.Equal(response.url, r.URL.RequestURI())

				w.WriteHeader(response.status)
				fmt.Fprintln(w, response.body)
			}))

			client := NewClient(ts.URL, MainnetGenesisBlockIdentifier, MainnetCurrency)
			peers, err := client.GetPeers(context.Background())
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedPeers, peers)
			}
		})
	}
}

func TestGetRawBlock(t *testing.T) {
	tests := map[string]struct {
		blockIdentifier *types.PartialBlockIdentifier
		responses       []responseFixture

		expectedBlock *Block
		expectedCoins []string
		expectedError error
	}{
		"lookup by hash": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Hash: &blockIdentifier1000.Hash,
			},
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response.json"),
					url:    url,
				},
			},
			expectedBlock: block1000,
			expectedCoins: []string{},
		},
		"lookup by hash 2": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Hash: &blockIdentifier100000.Hash,
			},
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response_2.json"),
					url:    url,
				},
			},
			expectedBlock: block100000,
			expectedCoins: []string{
				"87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03:0",
				"503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:0",
				"503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:1",
			},
		},
		"lookup by hash (get block api error)": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Hash: &blockIdentifier1000.Hash,
			},
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_not_found_response.json"),
					url:    url,
				},
			},
			expectedError: ErrBlockNotFound,
		},
		"lookup by hash (get block internal error)": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Hash: &blockIdentifier1000.Hash,
			},
			responses: []responseFixture{
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedBlock: nil,
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
		"lookup by index": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Index: &blockIdentifier1000.Index,
			},
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_hash_response.json"),
					url:    url,
				},
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response.json"),
					url:    url,
				},
			},
			expectedBlock: block1000,
			expectedCoins: []string{},
		},
		"lookup by index (out of range)": {
			blockIdentifier: &types.PartialBlockIdentifier{
				Index: &blockIdentifier1000.Index,
			},
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_hash_out_of_range_response.json"),
					url:    url,
				},
			},
			expectedError: errors.New("height out of range"),
		},
		"current block lookup": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("get_blockchain_info_response.json"),
					url:    url,
				},
				{
					status: http.StatusOK,
					body:   loadFixture("get_block_response.json"),
					url:    url,
				},
			},
			expectedBlock: block1000,
			expectedCoins: []string{},
		},
		"current block lookup (can't get current info)": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("rpc_in_warmup_response.json"),
					url:    url,
				},
			},
			expectedError: errors.New("unable to get blockchain info"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			responses := make(chan responseFixture, len(test.responses))
			for _, response := range test.responses {
				responses <- response
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := <-responses
				assert.Equal("application/json", r.Header.Get("Content-Type"))
				assert.Equal("POST", r.Method)
				assert.Equal(response.url, r.URL.RequestURI())

				w.WriteHeader(response.status)
				fmt.Fprintln(w, response.body)
			}))

			client := NewClient(ts.URL, MainnetGenesisBlockIdentifier, MainnetCurrency)
			block, coins, err := client.GetRawBlock(context.Background(), test.blockIdentifier)
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedBlock, block)
				assert.Equal(test.expectedCoins, coins)
			}
		})
	}
}

func int64Pointer(v int64) *int64 {
	return &v
}

func mustMarshalMap(v interface{}) map[string]interface{} {
	m, _ := types.MarshalMap(v)
	return m
}

func TestParseBlock(t *testing.T) {
	tests := map[string]struct {
		block *Block
		coins map[string]*types.AccountCoin

		expectedBlock *types.Block
		expectedError error
	}{
		"no fetched transactions": {
			block: block1000,
			coins: map[string]*types.AccountCoin{},
			expectedBlock: &types.Block{
				BlockIdentifier: blockIdentifier1000,
				ParentBlockIdentifier: &types.BlockIdentifier{
					Hash:  "0000000008e647742775a230787d66fdf92c46a48c896bfbc85cdc8acc67e87d",
					Index: 999,
				},
				Timestamp: 1232346882000,
				Transactions: []*types.Transaction{
					{
						TransactionIdentifier: &types.TransactionIdentifier{
							Hash: "fe28050b93faea61fa88c4c630f0e1f0a1c24d0082dd0e10d369e13212128f33",
						},
						Operations: []*types.Operation{
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        0,
									NetworkIndex: int64Pointer(0),
								},
								Type:   CoinbaseOpType,
								Status: types.String(SuccessStatus),
								Metadata: mustMarshalMap(&OperationMetadata{
									Coinbase: "04ffff001d02fd04",
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        1,
									NetworkIndex: int64Pointer(0),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "4104f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446aac", // nolint
								},
								Amount: &types.Amount{
									Value:    "5000000000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "fe28050b93faea61fa88c4c630f0e1f0a1c24d0082dd0e10d369e13212128f33:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:  "04f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446a OP_CHECKSIG", // nolint
										Hex:  "4104f5eeb2b10c944c6b9fbcfff94c35bdeecd93df977882babc7f3a2cf7f5c81d3b09a68db7f0e04f21de5d4230e75e6dbe7ad16eefe0d4325a62067dc6f369446aac",         // nolint
										Type: "pubkey",
									},
								}),
							},
						},
						Metadata: mustMarshalMap(&TransactionMetadata{
							Size:    135,
							Version: 1,
							Vsize:   135,
							Weight:  540,
						}),
					},
					{
						TransactionIdentifier: &types.TransactionIdentifier{
							Hash: "4852fe372ff7534c16713b3146bbc1e86379c70bea4d5c02fb1fa0112980a081",
						},
						Operations: []*types.Operation{
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        0,
									NetworkIndex: int64Pointer(0),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "mmtKKnjqTPdkBnBMbNt5Yu2SCwpMaEshEL", // nolint
								},
								Amount: &types.Amount{
									Value:    "3810000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "4852fe372ff7534c16713b3146bbc1e86379c70bea4d5c02fb1fa0112980a081:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:          "OP_DUP OP_HASH160 45db0b779c0b9fa207f12a8218c94fc77aff5045 OP_EQUALVERIFY OP_CHECKSIG", // nolint
										Hex:          "76a91445db0b779c0b9fa207f12a8218c94fc77aff504588ac",                                    // nolint
										Type:         "pubkeyhash",
										RequiredSigs: 1,
										Addresses: []string{
											"mmtKKnjqTPdkBnBMbNt5Yu2SCwpMaEshEL",
										},
									},
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        1,
									NetworkIndex: int64Pointer(1),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "4852fe372ff7534c16713b3146bbc1e86379c70bea4d5c02fb1fa0112980a081:1",
								},
								Amount: &types.Amount{
									Value:    "50000000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "4852fe372ff7534c16713b3146bbc1e86379c70bea4d5c02fb1fa0112980a081:1",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:  "",
										Hex:  "",
										Type: "nonstandard",
									},
								}),
							},
						},
						Metadata: mustMarshalMap(&TransactionMetadata{
							Size:    1408,
							Version: 1,
							Vsize:   1408,
							Weight:  5632,
						}),
					},
				},
				Metadata: mustMarshalMap(&BlockMetadata{
					Size:       216,
					Weight:     864,
					Version:    1,
					MerkleRoot: "fe28050b93faea61fa88c4c630f0e1f0a1c24d0082dd0e10d369e13212128f33",
					MedianTime: 1232344831,
					Nonce:      2595206198,
					Bits:       "1d00ffff",
					Difficulty: 1,
				}),
			},
		},
		"block 100000": {
			block: block100000,
			coins: map[string]*types.AccountCoin{
				"87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03:0": {
					Account: &types.AccountIdentifier{
						Address: "1BNwxHGaFbeUBitpjy2AsKpJ29Ybxntqvb",
					},
					Coin: &types.Coin{
						CoinIdentifier: &types.CoinIdentifier{
							Identifier: "87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03:0",
						},
						Amount: &types.Amount{
							Value:    "5000000000",
							Currency: MainnetCurrency,
						},
					},
				},
				"503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:0": {
					Account: &types.AccountIdentifier{
						Address: "3FfQGY7jqsADC7uTVqF3vKQzeNPiBPTqt4",
					},
					Coin: &types.Coin{
						CoinIdentifier: &types.CoinIdentifier{
							Identifier: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:0",
						},
						Amount: &types.Amount{
							Value:    "3467607",
							Currency: MainnetCurrency,
						},
					},
				},
				"503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:1": {
					Account: &types.AccountIdentifier{
						Address: "1NdvAyRJLdK5EXs7DV3ebYb5wffdCZk1pD",
					},
					Coin: &types.Coin{
						CoinIdentifier: &types.CoinIdentifier{
							Identifier: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:1",
						},
						Amount: &types.Amount{
							Value:    "0",
							Currency: MainnetCurrency,
						},
					},
				},
			},
			expectedBlock: &types.Block{
				BlockIdentifier: blockIdentifier100000,
				ParentBlockIdentifier: &types.BlockIdentifier{
					Hash:  "000000000002d01c1fccc21636b607dfd930d31d01c3a62104612a1719011250",
					Index: 99999,
				},
				Timestamp: 1293623863000,
				Transactions: []*types.Transaction{
					{
						TransactionIdentifier: &types.TransactionIdentifier{
							Hash: "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
						},
						Operations: []*types.Operation{
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        0,
									NetworkIndex: int64Pointer(0),
								},
								Type:   CoinbaseOpType,
								Status: types.String(SuccessStatus),
								Metadata: mustMarshalMap(&OperationMetadata{
									Coinbase: "044c86041b020602",
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        1,
									NetworkIndex: int64Pointer(0),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "34qkc2iac6RsyxZVfyE2S5U5WcRsbg2dpK",
								},
								Amount: &types.Amount{
									Value:    "1589351625",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:          "OP_HASH160 228f554bbf766d6f9cc828de1126e3d35d15e5fe OP_EQUAL",
										Hex:          "a914228f554bbf766d6f9cc828de1126e3d35d15e5fe87",
										RequiredSigs: 1,
										Type:         "scripthash",
										Addresses: []string{
											"34qkc2iac6RsyxZVfyE2S5U5WcRsbg2dpK",
										},
									},
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        2,
									NetworkIndex: int64Pointer(1),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "6a24aa21a9ed10109f4b82aa3ed7ec9d02a2a90246478b3308c8b85daf62fe501d58d05727a4",
								},
								Amount: &types.Amount{
									Value:    "0",
									Currency: MainnetCurrency,
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:  "OP_RETURN aa21a9ed10109f4b82aa3ed7ec9d02a2a90246478b3308c8b85daf62fe501d58d05727a4",
										Hex:  "6a24aa21a9ed10109f4b82aa3ed7ec9d02a2a90246478b3308c8b85daf62fe501d58d05727a4",
										Type: "nulldata",
									},
								}),
							},
						},
						Metadata: mustMarshalMap(&TransactionMetadata{
							Size:    135,
							Version: 1,
							Vsize:   135,
							Weight:  540,
						}),
					},
					{
						TransactionIdentifier: &types.TransactionIdentifier{
							Hash: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
						},
						Operations: []*types.Operation{
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        0,
									NetworkIndex: int64Pointer(0),
								},
								Type:   InputOpType,
								Status: types.String(SuccessStatus),
								Amount: &types.Amount{
									Value:    "-5000000000",
									Currency: MainnetCurrency,
								},
								Account: &types.AccountIdentifier{
									Address: "1BNwxHGaFbeUBitpjy2AsKpJ29Ybxntqvb",
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinSpent,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "87a157f3fd88ac7907c05fc55e271dc4acdc5605d187d646604ca8c0e9382e03:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptSig: &ScriptSig{
										ASM: "3046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748[ALL] 04f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
										Hex: "493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
									},
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        1,
									NetworkIndex: int64Pointer(0),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
								},
								Amount: &types.Amount{
									Value:    "556000000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:          "OP_DUP OP_HASH160 c398efa9c392ba6013c5e04ee729755ef7f58b32 OP_EQUALVERIFY OP_CHECKSIG",
										Hex:          "76a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac",
										RequiredSigs: 1,
										Type:         "pubkeyhash",
										Addresses: []string{
											"1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
										},
									},
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        2,
									NetworkIndex: int64Pointer(1),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "1EYTGtG4LnFfiMvjJdsU7GMGCQvsRSjYhx",
								},
								Amount: &types.Amount{
									Value:    "4444000000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4:1",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:          "OP_DUP OP_HASH160 948c765a6914d43f2a7ac177da2c2f6b52de3d7c OP_EQUALVERIFY OP_CHECKSIG",
										Hex:          "76a914948c765a6914d43f2a7ac177da2c2f6b52de3d7c88ac",
										RequiredSigs: 1,
										Type:         "pubkeyhash",
										Addresses: []string{
											"1EYTGtG4LnFfiMvjJdsU7GMGCQvsRSjYhx",
										},
									},
								}),
							},
						},
						Metadata: mustMarshalMap(&TransactionMetadata{
							Size:    259,
							Version: 1,
							Vsize:   259,
							Weight:  1036,
						}),
					},
					{
						TransactionIdentifier: &types.TransactionIdentifier{
							Hash: "fake",
						},
						Operations: []*types.Operation{
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        0,
									NetworkIndex: int64Pointer(0),
								},
								Type:   InputOpType,
								Status: types.String(SuccessStatus),
								Amount: &types.Amount{
									Value:    "-3467607",
									Currency: MainnetCurrency,
								},
								Account: &types.AccountIdentifier{
									Address: "3FfQGY7jqsADC7uTVqF3vKQzeNPiBPTqt4",
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinSpent,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptSig: &ScriptSig{
										ASM: "00142b2296c588ec413cebd19c3cbc04ea830ead6e78",
										Hex: "1600142b2296c588ec413cebd19c3cbc04ea830ead6e78",
									},
									TxInWitness: []string{
										"304402205f39ccbab38b644acea0776d18cb63ce3e37428cbac06dc23b59c61607aef69102206b8610827e9cb853ea0ba38983662034bd3575cc1ab118fb66d6a98066fa0bed01", // nolint
										"0304c01563d46e38264283b99bb352b46e69bf132431f102d4bd9a9d8dab075e7f",
									},
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        1,
									NetworkIndex: int64Pointer(1),
								},
								Type:   InputOpType,
								Status: types.String(SuccessStatus),
								Amount: &types.Amount{
									Value:    "0",
									Currency: MainnetCurrency,
								},
								Account: &types.AccountIdentifier{
									Address: "1NdvAyRJLdK5EXs7DV3ebYb5wffdCZk1pD",
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinSpent,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "503e4e9824282eb06f1a328484e2b367b5f4f93a405d6e7b97261bafabfb53d5:1",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptSig: &ScriptSig{
										ASM: "00142b2296c588ec413cebd19c3cbc04ea830ead6e78",
										Hex: "1600142b2296c588ec413cebd19c3cbc04ea830ead6e78",
									},
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        2,
									NetworkIndex: int64Pointer(2),
								},
								Type:   InputOpType,
								Status: types.String(SuccessStatus),
								Amount: &types.Amount{
									Value:    "-556000000",
									Currency: MainnetCurrency,
								},
								Account: &types.AccountIdentifier{
									Address: "1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinSpent,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptSig: &ScriptSig{
										ASM: "3046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748[ALL] 04f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
										Hex: "493046022100c352d3dd993a981beba4a63ad15c209275ca9470abfcd57da93b58e4eb5dce82022100840792bc1f456062819f15d33ee7055cf7b5ee1af1ebcc6028d9cdb1c3af7748014104f46db5e9d61a9dc27b8d64ad23e7383a4e6ca164593c2527c038c0857eb67ee8e825dca65046b82c9331586c82e0fd1f633f25f87c161bc6f8a630121df2b3d3", // nolint
									},
									Sequence: 4294967295,
								}),
							},
							{
								OperationIdentifier: &types.OperationIdentifier{
									Index:        3,
									NetworkIndex: int64Pointer(0),
								},
								Type:   OutputOpType,
								Status: types.String(SuccessStatus),
								Account: &types.AccountIdentifier{
									Address: "76a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac",
								},
								Amount: &types.Amount{
									Value:    "20056000000",
									Currency: MainnetCurrency,
								},
								CoinChange: &types.CoinChange{
									CoinAction: types.CoinCreated,
									CoinIdentifier: &types.CoinIdentifier{
										Identifier: "fake:0",
									},
								},
								Metadata: mustMarshalMap(&OperationMetadata{
									ScriptPubKey: &ScriptPubKey{
										ASM:          "OP_DUP OP_HASH160 c398efa9c392ba6013c5e04ee729755ef7f58b32 OP_EQUALVERIFY OP_CHECKSIG",
										Hex:          "76a914c398efa9c392ba6013c5e04ee729755ef7f58b3288ac",
										RequiredSigs: 1,
										Type:         "pubkeyhash",
										Addresses: []string{
											"1JqDybm2nWTENrHvMyafbSXXtTk5Uv5QAn",
											"1EYTGtG4LnFfiMvjJdsU7GMGCQvsRSjYhx",
										},
									},
								}),
							},
						},
						Metadata: mustMarshalMap(&TransactionMetadata{
							Size:     421,
							Version:  2,
							Vsize:    612,
							Weight:   129992,
							Locktime: 10,
						}),
					},
				},
				Metadata: mustMarshalMap(&BlockMetadata{
					Size:       957,
					Weight:     3828,
					Version:    1,
					MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
					MedianTime: 1293622620,
					Nonce:      274148111,
					Bits:       "1b04864c",
					Difficulty: 14484.1623612254,
				}),
			},
		},
		"missing transactions": {
			block:         block100000,
			coins:         map[string]*types.AccountCoin{},
			expectedError: errors.New("error finding previous tx"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			client := NewClient("", MainnetGenesisBlockIdentifier, MainnetCurrency)
			block, err := client.ParseBlock(context.Background(), test.block, test.coins)
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedBlock, block)
			}
		})
	}
}

func TestSuggestedFeeRate(t *testing.T) {
	tests := map[string]struct {
		responses []responseFixture

		expectedRate  float64
		expectedError error
	}{
		"successful": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("fee_rate.json"),
					url:    url,
				},
			},
			expectedRate: float64(0.00001),
		},
		"invalid range error": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("invalid_fee_rate.json"),
					url:    url,
				},
			},
			expectedError: errors.New("error getting fee estimate"),
		},
		"500 error": {
			responses: []responseFixture{
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			responses := make(chan responseFixture, len(test.responses))
			for _, response := range test.responses {
				responses <- response
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := <-responses
				assert.Equal("application/json", r.Header.Get("Content-Type"))
				assert.Equal("POST", r.Method)
				assert.Equal(response.url, r.URL.RequestURI())

				w.WriteHeader(response.status)
				fmt.Fprintln(w, response.body)
			}))

			client := NewClient(ts.URL, MainnetGenesisBlockIdentifier, MainnetCurrency)
			rate, err := client.SuggestedFeeRate(context.Background(), 1)
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedRate, rate)
			}
		})
	}
}

func TestRawMempool(t *testing.T) {
	tests := map[string]struct {
		responses []responseFixture

		expectedTransactions []string
		expectedError        error
	}{
		"successful": {
			responses: []responseFixture{
				{
					status: http.StatusOK,
					body:   loadFixture("raw_mempool.json"),
					url:    url,
				},
			},
			expectedTransactions: []string{
				"9cec12d170e97e21a876fa2789e6bfc25aa22b8a5e05f3f276650844da0c33ab",
				"37b4fcc8e0b229412faeab8baad45d3eb8e4eec41840d6ac2103987163459e75",
				"7bbb29ae32117597fcdf21b464441abd571dad52d053b9c2f7204f8ea8c4762e",
			},
		},
		"500 error": {
			responses: []responseFixture{
				{
					status: http.StatusInternalServerError,
					body:   "{}",
					url:    url,
				},
			},
			expectedError: errors.New("invalid response: 500 Internal Server Error"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				assert = assert.New(t)
			)

			responses := make(chan responseFixture, len(test.responses))
			for _, response := range test.responses {
				responses <- response
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := <-responses
				assert.Equal("application/json", r.Header.Get("Content-Type"))
				assert.Equal("POST", r.Method)
				assert.Equal(response.url, r.URL.RequestURI())

				w.WriteHeader(response.status)
				fmt.Fprintln(w, response.body)
			}))

			client := NewClient(ts.URL, MainnetGenesisBlockIdentifier, MainnetCurrency)
			txs, err := client.RawMempool(context.Background())
			if test.expectedError != nil {
				assert.Contains(err.Error(), test.expectedError.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedTransactions, txs)
			}
		})
	}
}

// loadFixture takes a file name and returns the response fixture.
func loadFixture(fileName string) string {
	content, err := ioutil.ReadFile(fmt.Sprintf("client_fixtures/%s", fileName))
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

type responseFixture struct {
	status int
	body   string
	url    string
}

<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
   Rosetta Bitcoin
</h3>
<p align="center">
  <a href="https://circleci.com/gh/coinbase/rosetta-bitcoin/tree/master"><img src="https://circleci.com/gh/coinbase/rosetta-bitcoin/tree/master.svg?style=shield" /></a>
  <a href="https://coveralls.io/github/coinbase/rosetta-bitcoin"><img src="https://coveralls.io/repos/github/coinbase/rosetta-bitcoin/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/coinbase/rosetta-bitcoin"><img src="https://goreportcard.com/badge/github.com/coinbase/rosetta-bitcoin" /></a>
  <a href="https://github.com/coinbase/rosetta-bitcoin/blob/master/LICENSE.txt"><img src="https://img.shields.io/github/license/coinbase/rosetta-bitcoin.svg" /></a>
  <a href="https://pkg.go.dev/github.com/coinbase/rosetta-bitcoin?tab=overview"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=shield" /></a>
</p>

<p align="center"><b>
ROSETTA-BITCOIN IS CONSIDERED <a href="https://en.wikipedia.org/wiki/Software_release_life_cycle#Alpha">ALPHA SOFTWARE</a>.
USE AT YOUR OWN RISK! COINBASE ASSUMES NO RESPONSIBILITY NOR LIABILITY IF THERE IS A BUG IN THIS IMPLEMENTATION.
</b></p>

## Overview
`rosetta-bitcoin` provides a reference implementation of the Rosetta API for
Bitcoin in Golang. If you haven't heard of the Rosetta API, you can find more
information [here](https://rosetta-api.org).

## Features
* Rosetta API implementation (both Data API and Construction API)
* UTXO cache for all accounts (accessible using `/account/balance`)
* Stateless, offline, curve-based transaction construction from any SegWit-Bech32 Address

## Usage
As specified in the [Rosetta API Principles](https://www.rosetta-api.org/docs/automated_deployment.html),
all Rosetta implementations must be deployable via Docker and support running via either an
[`online` or `offline` mode](https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes).

To build a Docker image from this repository, run the command `make build`. To start
`rosetta-bitcoin`, you can run:
* `make run-mainnet-online`
* `make run-mainnet-offline`
* `make run-testnet-online`
* `make run testnet-offline`

By default, running these commands will create a data directory at `<working directory>/bitcoin-data`
and start the `rosetta-bitcoin` server at port `8080`.

## System Requirements
`rosetta-bitcoin` has been tested on an [AWS c5.2xlarge instance](https://aws.amazon.com/ec2/instance-types/c5).
This instance type has 8 vCPU and 16 GB of RAM. If you use a computer with less than 16 GB of RAM,
it is possible that `rosetta-bitcoin` will exit with an OOM error.

### Recommended OS Settings
To increase the load `rosetta-bitcoin` can handle, it is recommended to tune your OS
settings to allow for more connections. On a linux-based OS, you can run the following
commands ([source](http://www.tweaked.io/guide/kernel)):
```text
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w net.core.somaxconn=10000
sysctl -p (when done)
```
_We have not tested `rosetta-bitcoin` with `net.ipv4.tcp_tw_recycle` and do not recommend
enabling it._

You should also modify your open file settings to `100000`. This can be done on a linux-based OS
with the command: `ulimit -n 100000`.

### Optimizations
* Automatically prune bitcoind while indexing blocks
* Reduce sync time with concurrent block indexing
* Use [Zstandard compression](https://github.com/facebook/zstd) to reduce the size of data stored on disk
without needing to write a manual byte-level encoding

#### Concurrent Block Syncing
To speed up indexing, `rosetta-bitcoin` uses concurrent block processing
with a "wait free" design (using channels instead of sleeps to signal
which threads are unblocked). This allows `rosetta-bitcoin` to fetch
multiple inputs from disk while it waits for inputs that appeared
in recently processed blocks to save to disk.
```text
                                                   +----------+
                                                   | bitcoind |
                                                   +-----+----+
                                                         |
                                                         |
          +---------+ fetch block data / unpopulated txs |
          | block 1 <------------------------------------+
          +---------+                                    |
       +-->   tx 1  |                                    |
       |  +---------+                                    |
       |  |   tx 2  |                                    |
       |  +----+----+                                    |
       |       |                                         |
       |       |           +---------+                   |
       |       |           | block 2 <-------------------+
       |       |           +---------+                   |
       |       +----------->   tx 3  +--+                |
       |                   +---------+  |                |
       +------------------->   tx 4  |  |                |
       |                   +---------+  |                |
       |                                |                |
       | retrieve previously synced     |   +---------+  |
       | inputs needed for future       |   | block 3 <--+
       | blocks while waiting for       |   +---------+
       | populated blocks to save to    +--->   tx 5  |
       | disk                               +---------+
       +------------------------------------>   tx 6  |
       |                                    +---------+
       |
       |
+------+--------+
|  coin_storage |
+---------------+
```

### Architecture
`rosetta-bitcoin` uses the `syncer`, `storage`, `parser`, and `server` package
from [`rosetta-sdk-go`](https://github.com/coinbase/rosetta-sdk-go) instead
of a new Bitcoin-specific implementation of packages of similar functionality. Below
you can find a high-level overview of how everything fits together:
```text
                               +------------------------------------------------------------------+
                               |                                                                  |
                               |                 +--------------------------------------+         |
                               |                 |                                      |         |
                               |                 |                 indexer              |         |
                               |                 |                                      |         |
                               |                 | +--------+                           |         |
                               +-------------------+ pruner <----------+                |         |
                               |                 | +--------+          |                |         |
                         +-----v----+            |                     |                |         |
                         | bitcoind |            |              +------+--------+       |         |
                         +-----+----+            |     +--------> block_storage <----+  |         |
                               |                 |     |        +---------------+    |  |         |
                               |                 | +---+----+                        |  |         |
                               +-------------------> syncer |                        |  |         |
                                                 | +---+----+                        |  |         |
                                                 |     |        +--------------+     |  |         |
                                                 |     +--------> coin_storage |     |  |         |
                                                 |              +------^-------+     |  |         |
                                                 |                     |             |  |         |
                                                 +--------------------------------------+         |
                                                                       |             |            |
+-------------------------------------------------------------------------------------------+     |
|                                                                      |             |      |     |
|         +------------------------------------------------------------+             |      |     |
|         |                                                                          |      |     |
|         |                     +---------------------+-----------------------+------+      |     |
|         |                     |                     |                       |             |     |
| +-------+---------+   +-------+---------+   +-------+-------+   +-----------+----------+  |     |
| | account_service |   | network_service |   | block_service |   | construction_service +--------+
| +-----------------+   +-----------------+   +---------------+   +----------------------+  |
|                                                                                           |
|                                         server                                            |
|                                                                                           |
+-------------------------------------------------------------------------------------------+
```

## Testing with rosetta-cli
To validate `rosetta-bitcoin`, [install `rosetta-cli`](https://github.com/coinbase/rosetta-cli#install)
and run one of the following commands:
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/bitcoin_testnet.json`
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/bitcoin_testnet.json`
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/bitcoin_mainnet.json`
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/bitcoin_mainnet.json`

## Future Work
* Publish benchamrks for sync speed, storage usage, and load testing
* Rosetta API `/mempool/*` implementation
* Add CI test using `rosetta-cli` to run on each PR (likely on a regtest network)
* Add performance mode to use unlimited RAM (implementation currently optimized to use <= 16 GB of RAM)
* Support Multi-Sig Sends

_Please reach out on our [community](https://community.rosetta-api.org) if you want to tackle anything on this list!_

## Development
* `make deps` to install dependencies
* `make test` to run tests
* `make lint` to lint the source code
* `make salus` to check for security concerns
* `make build-local` to build a Docker image from the local context
* `make coverage-local` to generate a coverage report

## License
This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2020 Coinbase

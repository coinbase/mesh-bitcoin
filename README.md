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
USE AT YOUR OWN RISK.</b><p>
<p align="center">This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).</p>

## Overview

The `rosetta-bitcoin` repository provides a reference implementation of the Rosetta API for Bitcoin in Golang. This repository was created for developers of Bitcoin-like (a.k.a., UTXO) blockchains, who may find it easier to fork this reference implementation than write one from scratch.

[Rosetta](https://www.rosetta-api.org/docs/welcome.html) is an open-source specification and set of tools that makes integrating with blockchains simpler, faster, and more reliable. The Rosetta API is specified in the [OpenAPI 3.0 format](https://www.openapis.org).

Requests and responses can be crafted with auto-generated code using [Swagger Codegen](https://swagger.io/tools/swagger-codegen) or [OpenAPI Generator](https://openapi-generator.tech), are human-readable (easy to debug and understand), and can be used in servers and browsers.

## Features

* Rosetta API implementation (both Data API and Construction API)
* UTXO cache for all accounts (accessible using the Rosetta `/account/balance` API)
* Stateless, offline, curve-based transaction construction from any SegWit-Bech32 Address
* Automatically prune bitcoind while indexing blocks
* Reduce sync time with concurrent block indexing
* Use [Zstandard compression](https://github.com/facebook/zstd) to reduce the size of data stored on disk without needing to write a manual byte-level encoding

## System Requirements

The `rosetta-bitcoin` implementation has been tested on an [AWS c5.2xlarge instance](https://aws.amazon.com/ec2/instance-types/c5). This instance type has 8 vCPU and 16 GB of RAM.

## Getting Started

1. Adjust your [network settings](#network-settings) to the recommended connections.
2. Install and run Docker as directed in the [Deployment](#deployment) section below.
3. Run the [`Testnet:Online`](#testnetonline) command.

### Network Settings

To increase the load that `rosetta-bitcoin` can handle, we recommend tunning your OS settings to allow for more connections. On a linux-based OS, you can run these commands ([source](http://www.tweaked.io/guide/kernel)):

```text
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w net.core.somaxconn=10000
sysctl -p (when done)
```
_We have not tested `rosetta-bitcoin` with `net.ipv4.tcp_tw_recycle` and do not recommend enabling it._

You should also modify your open file settings to `100000`. This can be done on a linux-based OS with the command: `ulimit -n 100000`.

### Memory-Mapped Files

`rosetta-bitcoin` uses [memory-mapped files](https://en.wikipedia.org/wiki/Memory-mapped_file) to persist data in the `indexer`. As a result, you **must** run `rosetta-bitcoin` on a 64-bit architecture (the virtual address space easily exceeds 100s of GBs).

If you receive a kernel OOM, you may need to increase the allocated size of swap space on your OS. There is a great tutorial for how to do this on Linux [here](https://linuxize.com/post/create-a-linux-swap-file/).

## Development

While working on improvements to this repository, we recommend that you use these commands to check your code:

* `make deps` to install dependencies
* `make test` to run tests
* `make lint` to lint the source code
* `make salus` to check for security concerns
* `make build-local` to build a Docker image from the local context
* `make coverage-local` to generate a coverage report

### Deployment

As specified in the [Rosetta API Principles](https://www.rosetta-api.org/docs/automated_deployment.html), all Rosetta implementations must be deployable via Docker and support running via either an [`online` or `offline` mode](https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes).

**YOU MUST [INSTALL DOCKER](https://www.docker.com/get-started) FOR THESE INSTRUCTIONS TO WORK.**

#### Image Installation

Running these commands will create a Docker image called `rosetta-bitcoin:latest`.

##### Installing from GitHub

To download the pre-built Docker image from the latest release, run:

```text
curl -sSfL https://raw.githubusercontent.com/coinbase/rosetta-bitcoin/master/install.sh | sh -s
```
_Do not try to install rosetta-bitcoin using GitHub Packages!_

##### Installing from Source

After cloning this repository, run:

```text
make build-local
```

#### Run Docker

Running these commands will start a Docker container in [detached mode](https://docs.docker.com/engine/reference/run/#detached--d) with a data directory at `<working directory>/bitcoin-data` and the Rosetta API accessible at port `8080`.

##### Required Arguments

**`MODE`** 
**Type:** `String`
**Options:** `ONLINE`, `OFFLINE`
**Default:** None

`MODE` determines if Rosetta can make outbound connections.

**`NETWORK`**
**Type:** `String`
**Options:** `MAINNET`, `ROPSTEN`, `RINKEBY`, `GOERLI` or `TESTNET`
**Default:** `ROPSTEN`, but only for backwards compatibility if you use `TESTNET`

`NETWORK` is the Ethereum network to launch or communicate with.

**`PORT`**
**Type:** `Integer`
**Options:** `8080`, any compatible port number
**Default:** None

`PORT` is the port to use for Rosetta.

##### Command Examples

You can run these commands from the command line. If you cloned the repository, you can use the `make` commands shown after the examples.

###### **Mainnet:Online**

Uncloned repo:
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/bitcoin-data:/data" -e "MODE=ONLINE" -e "NETWORK=MAINNET" -e "PORT=8080" -p 8080:8080 -p 8333:8333 rosetta-bitcoin:latest
```
Cloned repo:
```text
make run-mainnet-online
```

###### **Mainnet:Offline**

Uncloned repo:
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=MAINNET" -e "PORT=8081" -p 8081:8081 rosetta-bitcoin:latest
```
Cloned repo:
```text
make run-mainnet-offline
```

###### **Testnet:Online**

Uncloned repo:
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/bitcoin-data:/data" -e "MODE=ONLINE" -e "NETWORK=TESTNET" -e "PORT=8080" -p 8080:8080 -p 18333:18333 rosetta-bitcoin:latest
```

Cloned repo: 
```text
make run-testnet-online
```

###### **Testnet:Offline**

Uncloned repo:
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=TESTNET" -e "PORT=8081" -p 8081:8081 rosetta-bitcoin:latest
```

Cloned repo: 
```text
make run-testnet-offline
```

## Architecture

`rosetta-bitcoin` uses the `syncer`, `storage`, `parser`, and `server` package from [`rosetta-sdk-go`](https://github.com/coinbase/rosetta-sdk-go) instead of a new Bitcoin-specific implementation of packages of similar functionality. Below you can find an overview of how everything fits together:

<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Architecture" src="https://www.rosetta-api.org/img/rosetta_bitcoin_architecture.jpg">
  </a>
</p>

### Concurrent Block Syncing

To speed up indexing, `rosetta-bitcoin` uses concurrent block processing with a "wait free" design (using [the channels function](https://golangdocs.com/channels-in-golang) instead of [the sleep function](https://pkg.go.dev/time#Sleep) to signal which threads are unblocked). This allows `rosetta-bitcoin` to fetch multiple inputs from disk while it waits for inputs that appeared in recently processed blocks to save to disk.

<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Concurrent Block Syncing" src="https://www.rosetta-api.org/img/rosetta_bitcoin_concurrent_block_synching.jpg">
  </a>
</p>

## Test the Implementation with the rosetta-cli Tool

To validate `rosetta-bitcoin`, [install `rosetta-cli`](https://github.com/coinbase/rosetta-cli#install) and run one of these commands:

* `rosetta-cli check:data --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates that the Data API information in the `testnet` network is correct. It also ensures that the implementation does not miss any balance-changing operations.
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates the blockchain’s construction, signing, and broadcasting.
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/mainnet/config.json` - This command validates that the Data API information in the `mainnet` network is correct. It also ensures that the implementation does not miss any balance-changing operations.

Read the [How to Test your Rosetta Implementation](https://www.rosetta-api.org/docs/rosetta_test.html) documentation for additional details.

## Contributing

You may contribute to the `rosetta-bitcoin` project in various ways:

* [Asking Questions](CONTRIBUTING.md/#asking-questions)
* [Providing Feedback](CONTRIBUTING.md/#providing-feedback)
* [Reporting Issues](CONTRIBUTING.md/#reporting-issues)

Read our [Contributing](CONTRIBUTING.MD) documentation for more information.

When you've finished an implementation for a blockchain, share your work in the [ecosystem category of the community site](https://community.rosetta-api.org/c/ecosystem). Platforms looking for implementations for certain blockchains will be monitoring this section of the website for high-quality implementations they can use for integration. Make sure that your implementation meets the [expectations](https://www.rosetta-api.org/docs/node_deployment.html) of any implementation.

You can also find community implementations for a variety of blockchains in the [rosetta-ecosystem](https://github.com/coinbase/rosetta-ecosystem) repository.

## Documentation

You can find the Rosetta API documentation at [rosetta-api.org](https://www.rosetta-api.org/docs/welcome.html). 

Check out the [Getting Started](https://www.rosetta-api.org/docs/getting_started.html) section to start diving into Rosetta. 

Our documentation is divided into the following sections:

* [Product Overview](https://www.rosetta-api.org/docs/welcome.html)
* [Getting Started](https://www.rosetta-api.org/docs/getting_started.html)
* [Rosetta API Spec](https://www.rosetta-api.org/docs/Reference.html)
* [Testing](https://www.rosetta-api.org/docs/rosetta_cli.html)
* [Best Practices](https://www.rosetta-api.org/docs/node_deployment.html)
* [Repositories](https://www.rosetta-api.org/docs/rosetta_specifications.html)

## Related Projects

* [rosetta-sdk-go](https://github.com/coinbase/rosetta-sdk-go) — The `rosetta-sdk-go` SDK provides a collection of packages used for interaction with the Rosetta API specification. 
* [rosetta-specifications](https://github.com/coinbase/rosetta-specifications) — Much of the SDK code is generated from this repository.
* [rosetta-cli](https://github.com/coinbase/rosetta-ecosystem) — Use the `rosetta-cli` tool to test your Rosetta API implementation. The tool also provides the ability to look up block contents and account balances.

### Sample Implementations

You can find community implementations for a variety of blockchains in the [rosetta-ecosystem](https://github.com/coinbase/rosetta-ecosystem) repository, and in the [ecosystem category](https://community.rosetta-api.org/c/ecosystem) of our community site. 

## License
This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2022 Coinbase
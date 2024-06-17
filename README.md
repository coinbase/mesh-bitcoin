<p align="center">
  <a href="https://www.mesh-api.org">
    <img width="90%" alt="Mesh" src="https://www.mesh-api.org/img/mesh_header.png">
  </a>
</p>
<h3 align="center">
   Mesh Bitcoin
</h3>
<p align="center">
  <a href="https://circleci.com/gh/coinbase/mesh-bitcoin/tree/master"><img src="https://circleci.com/gh/coinbase/mesh-bitcoin/tree/master.svg?style=shield" /></a>
  <a href="https://coveralls.io/github/coinbase/mesh-bitcoin"><img src="https://coveralls.io/repos/github/coinbase/mesh-bitcoin/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/coinbase/mesh-bitcoin"><img src="https://goreportcard.com/badge/github.com/coinbase/mesh-bitcoin" /></a>
  <a href="https://github.com/coinbase/mesh-bitcoin/blob/master/LICENSE.txt"><img src="https://img.shields.io/github/license/coinbase/mesh-bitcoin.svg" /></a>
  <a href="https://pkg.go.dev/github.com/coinbase/mesh-bitcoin?tab=overview"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=shield" /></a>
</p>

<p align="center"><b>
MESH-BITCOIN IS CONSIDERED <a href="https://en.wikipedia.org/wiki/Software_release_life_cycle#Alpha">ALPHA SOFTWARE</a>.
USE AT YOUR OWN RISK.</b><p>
<p align="center">This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).</p>

## Overview

The `mesh-bitcoin` repository provides a reference implementation of the Mesh API for Bitcoin in Golang. This repository was created for developers of Bitcoin-like (a.k.a., UTXO) blockchains, who may find it easier to fork this reference implementation than write one from scratch.

[Mesh](https://www.mesh-api.org/docs/welcome.html) is an open-source specification and set of tools that makes integrating with blockchains simpler, faster, and more reliable. The Mesh API is specified in the [OpenAPI 3.0 format](https://www.openapis.org).

Requests and responses can be crafted with auto-generated code using [Swagger Codegen](https://swagger.io/tools/swagger-codegen) or [OpenAPI Generator](https://openapi-generator.tech), are human-readable (easy to debug and understand), and can be used in servers and browsers.

## Features

* Mesh API implementation (both Data API and Construction API)
* UTXO cache for all accounts (accessible using the Mesh `/account/balance` API)
* Stateless, offline, curve-based transaction construction from any SegWit-Bech32 Address
* Automatically prune bitcoind while indexing blocks
* Reduce sync time with concurrent block indexing
* Use [Zstandard compression](https://github.com/facebook/zstd) to reduce the size of data stored on disk without needing to write a manual byte-level encoding

## System Requirements

The `mesh-bitcoin` implementation has been tested on an [AWS c5.2xlarge instance](https://aws.amazon.com/ec2/instance-types/c5). This instance type has 8 vCPU and 16 GB of RAM.

## Getting Started

1. Adjust your [network settings](#network-settings) to the recommended connections.
2. Install and run Docker as directed in the [Deployment](#deployment) section below.
3. Run the [`Testnet:Online`](#testnetonline) command.

### Network Settings

To increase the load that `mesh-bitcoin` can handle, we recommend tunning your OS settings to allow for more connections. On a linux-based OS, you can run these commands ([source](http://www.tweaked.io/guide/kernel)):

```text
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w net.core.somaxconn=10000
sysctl -p (when done)
```
_We have not tested `mesh-bitcoin` with `net.ipv4.tcp_tw_recycle` and do not recommend enabling it._

You should also modify your open file settings to `100000`. This can be done on a linux-based OS with the command: `ulimit -n 100000`.

### Memory-Mapped Files

`mesh-bitcoin` uses [memory-mapped files](https://en.wikipedia.org/wiki/Memory-mapped_file) to persist data in the `indexer`. As a result, you **must** run `mesh-bitcoin` on a 64-bit architecture (the virtual address space easily exceeds 100s of GBs).

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

As specified in the [Mesh API Principles](https://www.mesh-api.org/docs/automated_deployment.html), all Mesh implementations must be deployable via Docker and support running via either an [`online` or `offline` mode](https://www.mesh-api.org/docs/node_deployment.html#multiple-modes).

**YOU MUST [INSTALL DOCKER](https://www.docker.com/get-started) FOR THESE INSTRUCTIONS TO WORK.**

#### Image Installation

Running these commands will create a Docker image called `mesh-bitcoin:latest`.

##### Installing from GitHub

To download the pre-built Docker image from the latest release, run:

```text
curl -sSfL https://raw.githubusercontent.com/coinbase/mesh-bitcoin/master/install.sh | sh -s
```
_Do not try to install mesh-bitcoin using GitHub Packages!_

##### Installing from Source

After cloning this repository, run:

```text
make build-local
```

#### Run Docker

Running these commands will start a Docker container in [detached mode](https://docs.docker.com/engine/reference/run/#detached--d) with a data directory at `<working directory>/bitcoin-data` and the Mesh API accessible at port `8080`.

##### Required Arguments

**`MODE`** - Determines whether Mesh can make outbound connections.
- **Type:** `String`
- **Options:** `ONLINE`, `OFFLINE`
- **Default:** None

**`NETWORK`** - The Ethereum network to launch or communicate with.
- **Type:** `String`
- **Options:** `MAINNET`, `ROPSTEN`, `RINKEBY`, `GOERLI` or `TESTNET`
- **Default:** `ROPSTEN`, but only for backwards compatibility if you use `TESTNET`

**`PORT`** - The port to use for Mesh.
- **Type:** `Integer`
- **Options:** `8080`, any compatible port number.
- **Default:** None

##### Command Examples

You can run these commands from the command line. If you cloned the repository, you can use the `make` commands shown after the examples.

###### **Mainnet:Online**

Uncloned repo:
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/bitcoin-data:/data" -e "MODE=ONLINE" -e "NETWORK=MAINNET" -e "PORT=8080" -p 8080:8080 -p 8333:8333 mesh-bitcoin:latest
```
Cloned repo:
```text
make run-mainnet-online
```

###### **Mainnet:Offline**

Uncloned repo:
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=MAINNET" -e "PORT=8081" -p 8081:8081 mesh-bitcoin:latest
```
Cloned repo:
```text
make run-mainnet-offline
```

###### **Testnet:Online**

Uncloned repo:
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/bitcoin-data:/data" -e "MODE=ONLINE" -e "NETWORK=TESTNET" -e "PORT=8080" -p 8080:8080 -p 18333:18333 mesh-bitcoin:latest
```

Cloned repo: 
```text
make run-testnet-online
```

###### **Testnet:Offline**

Uncloned repo:
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=TESTNET" -e "PORT=8081" -p 8081:8081 mesh-bitcoin:latest
```

Cloned repo: 
```text
make run-testnet-offline
```

## Architecture

`mesh-bitcoin` uses the `syncer`, `storage`, `parser`, and `server` package from [`mesh-sdk-go`](https://github.com/coinbase/mesh-sdk-go) instead of a new Bitcoin-specific implementation of packages of similar functionality. Below you can find an overview of how everything fits together:

<p align="center">
  <a href="https://www.mesh-api.org">
    <img width="90%" alt="Architecture" src="https://www.mesh-api.org/img/mesh_bitcoin_architecture.jpg">
  </a>
</p>

### Concurrent Block Syncing

To speed up indexing, `mesh-bitcoin` uses concurrent block processing with a "wait free" design (using [the channels function](https://golangdocs.com/channels-in-golang) instead of [the sleep function](https://pkg.go.dev/time#Sleep) to signal which threads are unblocked). This allows `mesh-bitcoin` to fetch multiple inputs from disk while it waits for inputs that appeared in recently processed blocks to save to disk.

<p align="center">
  <a href="https://www.mesh-api.org">
    <img width="90%" alt="Concurrent Block Syncing" src="https://www.mesh-api.org/img/mesh_bitcoin_concurrent_block_synching.jpg">
  </a>
</p>

## Test the Implementation with the mesh-cli Tool

To validate `mesh-bitcoin`, [install `mesh-cli`](https://github.com/coinbase/mesh-cli#install) and run one of these commands:

* `mesh-cli check:data --configuration-file mesh-cli-conf/testnet/config.json` - This command validates that the Data API information in the `testnet` network is correct. It also ensures that the implementation does not miss any balance-changing operations.
* `mesh-cli check:construction --configuration-file mesh-cli-conf/testnet/config.json` - This command validates the blockchain’s construction, signing, and broadcasting.
* `mesh-cli check:data --configuration-file mesh-cli-conf/mainnet/config.json` - This command validates that the Data API information in the `mainnet` network is correct. It also ensures that the implementation does not miss any balance-changing operations.

Read the [How to Test your Mesh Implementation](https://www.mesh-api.org/docs/mesh_test.html) documentation for additional details.

## Contributing

You may contribute to the `mesh-bitcoin` project in various ways:

* [Asking Questions](CONTRIBUTING.md/#asking-questions)
* [Providing Feedback](CONTRIBUTING.md/#providing-feedback)
* [Reporting Issues](CONTRIBUTING.md/#reporting-issues)

Read our [Contributing](CONTRIBUTING.MD) documentation for more information.

When you've finished an implementation for a blockchain, share your work in the [ecosystem category of the community site](https://community.mesh-api.org/c/ecosystem). Platforms looking for implementations for certain blockchains will be monitoring this section of the website for high-quality implementations they can use for integration. Make sure that your implementation meets the [expectations](https://www.mesh-api.org/docs/node_deployment.html) of any implementation.

You can also find community implementations for a variety of blockchains in the [mesh-ecosystem](https://github.com/coinbase/mesh-ecosystem) repository.

## Documentation

You can find the Mesh API documentation at [mesh-api.org](https://www.mesh-api.org/docs/welcome.html). 

Check out the [Getting Started](https://www.mesh-api.org/docs/getting_started.html) section to start diving into Mesh. 

Our documentation is divided into the following sections:

* [Product Overview](https://www.mesh-api.org/docs/welcome.html)
* [Getting Started](https://www.mesh-api.org/docs/getting_started.html)
* [Mesh API Spec](https://www.mesh-api.org/docs/Reference.html)
* [Testing](https://www.mesh-api.org/docs/mesh_cli.html)
* [Best Practices](https://www.mesh-api.org/docs/node_deployment.html)
* [Repositories](https://www.mesh-api.org/docs/mesh_specifications.html)

## Related Projects

* [mesh-sdk-go](https://github.com/coinbase/mesh-sdk-go) — The `mesh-sdk-go` SDK provides a collection of packages used for interaction with the Mesh API specification. 
* [mesh-specifications](https://github.com/coinbase/mesh-specifications) — Much of the SDK code is generated from this repository.
* [mesh-cli](https://github.com/coinbase/mesh-ecosystem) — Use the `mesh-cli` tool to test your Mesh API implementation. The tool also provides the ability to look up block contents and account balances.

### Sample Implementations

You can find community implementations for a variety of blockchains in the [mesh-ecosystem](https://github.com/coinbase/mesh-ecosystem) repository, and in the [ecosystem category](https://community.mesh-api.org/c/ecosystem) of our community site. 

## License
This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2022 Coinbase
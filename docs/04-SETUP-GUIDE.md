# Setup, Installation, and Running Guide

This guide covers everything needed to build and run the CometBFT Full-Stack POC.

---

## 1. Prerequisites

### Required (for all users)

| Tool              | Version   | Purpose                        | Install                                    |
|-------------------|-----------|--------------------------------|--------------------------------------------|
| Docker            | 20.10+    | Container runtime              | https://docs.docker.com/get-docker/        |
| Docker Compose    | 2.0+      | Multi-container orchestration  | Included with Docker Desktop               |
| curl              | any       | API testing                    | Pre-installed on most systems              |
| jq                | any       | JSON formatting                | `brew install jq` / `apt install jq`       |

### Optional (for local development)

| Tool              | Version   | Purpose                        | Install                                    |
|-------------------|-----------|--------------------------------|--------------------------------------------|
| Go                | 1.22+     | Build chain, API backend       | https://go.dev/dl/                         |
| Rust              | 1.75+     | Build smart contracts          | https://rustup.rs/                         |
| wasm32 target     | -         | Compile Rust to WASM           | `rustup target add wasm32-unknown-unknown` |
| buf               | latest    | Protobuf code generation       | https://buf.build/docs/installation        |
| golangci-lint     | latest    | Go linting                     | https://golangci-lint.run/usage/install/   |

---

## 2. Quick Start with Docker (Recommended)

### Step 1: Clone and Build

```bash
cd poc-cometbft
docker compose up --build -d
```

This builds three Docker images and starts three containers:
- `mychain-node` -- Cosmos SDK blockchain
- `abci-kvstore` -- Direct ABCI key-value store
- `api-backend` -- REST API server

### Step 2: Wait for Health

The chain needs ~30 seconds to initialize and start producing blocks.

```bash
# Watch container status (wait for all to show "healthy")
docker compose ps

# Or poll the health endpoint
curl http://localhost:8080/api/health
```

Expected healthy response:
```json
{
  "success": true,
  "data": {
    "chain": "ok",
    "kvstore": "ok"
  }
}
```

### Step 3: Deploy Smart Contract (Optional)

The bounty contract is NOT deployed automatically. To deploy it:

```bash
# Execute deploy script inside the mychain container
docker compose exec mychain-node bash /scripts/deploy-contracts.sh
```

What this does:
1. Waits for the chain to be ready
2. Uploads the compiled WASM file (`task_bounty.wasm`) to the chain
3. Gets Code ID 1
4. Instantiates the contract with `fee_pct: 10` (10% platform fee)
5. Prints the contract address

### Step 4: Verify Everything Works

```bash
# Health check
curl http://localhost:8080/api/health

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "My first task", "description": "Testing the blockchain"}'

# List tasks (wait ~5 seconds for the block to commit)
curl http://localhost:8080/api/tasks

# Set a key-value pair
curl -X PUT http://localhost:8080/api/kv/greeting \
  -H "Content-Type: application/json" \
  -d '{"value": "hello world"}'

# Read it back
curl http://localhost:8080/api/kv/greeting
```

---

## 3. Docker Container Topology

```
+------------------------------------------------------------------+
|  Docker Host                                                      |
|                                                                   |
|  docker compose up --build -d                                     |
|                                                                   |
|  +------------------------+     +------------------------+       |
|  | mychain-node           |     | abci-kvstore           |       |
|  | Image: mychain:latest  |     | Image: kvstore:latest  |       |
|  |                        |     |                        |       |
|  | Ports:                 |     | Ports:                 |       |
|  |   26657 -> 26657 (RPC) |     |   36657 -> 36657 (RPC) |       |
|  |   9090  -> 9090 (gRPC) |     |                        |       |
|  |   1317  -> 1317 (REST) |     | Volume: abci-data      |       |
|  |   26656 -> 26656 (P2P) |     +------------------------+       |
|  |                        |                                       |
|  | Volume: mychain-data   |     +------------------------+       |
|  +------------------------+     | api-backend            |       |
|                                 | Image: api:latest      |       |
|                                 |                        |       |
|                                 | Ports:                 |       |
|                                 |   8080 -> 8080 (HTTP)  |       |
|                                 |                        |       |
|                                 | Env:                   |       |
|                                 |  CHAIN_GRPC_ADDR=      |       |
|                                 |   mychain-node:9090    |       |
|                                 |  CHAIN_RPC_ADDR=       |       |
|                                 |   http://mychain-      |       |
|                                 |   node:26657           |       |
|                                 |  KVSTORE_RPC_ADDR=     |       |
|                                 |   http://abci-         |       |
|                                 |   kvstore:36657        |       |
|                                 +------------------------+       |
|                                                                   |
|  Network: cometbft-net (bridge)                                   |
+------------------------------------------------------------------+
```

---

## 4. Environment Variables Reference

These are set in `docker-compose.yml` for the `api-backend` service:

| Variable           | Default                         | Description                            |
|--------------------|---------------------------------|----------------------------------------|
| `CHAIN_GRPC_ADDR`  | `localhost:9090`                | gRPC endpoint for MyChain              |
| `CHAIN_RPC_ADDR`   | `http://localhost:26657`        | CometBFT RPC endpoint for MyChain      |
| `KVSTORE_RPC_ADDR` | `http://localhost:36657`        | CometBFT RPC endpoint for KVStore      |
| `CHAIN_ID`         | `mychain-1`                     | Chain identifier                       |
| `TEST_MNEMONIC`    | (12-word mnemonic)              | Mnemonic for the signing key           |
| `WASM_CODE_ID`     | `1`                             | Code ID of the deployed WASM contract  |
| `PORT`             | `8080`                          | API backend listen port                |

---

## 5. Port Reference

| Port  | Protocol | Service         | Purpose                              | Exposed |
|-------|----------|-----------------|--------------------------------------|---------|
| 8080  | HTTP     | API Backend     | Unified REST API                     | Yes     |
| 26657 | HTTP     | MyChain         | CometBFT RPC (tx broadcast, queries) | Yes     |
| 9090  | gRPC     | MyChain         | Cosmos SDK gRPC (typed queries)      | Yes     |
| 1317  | HTTP     | MyChain         | Cosmos SDK REST (Swagger)            | Yes     |
| 26656 | TCP      | MyChain         | P2P peer communication               | Yes     |
| 36657 | HTTP     | ABCI KVStore    | CometBFT RPC for KVStore             | Yes     |
| 36656 | TCP      | ABCI KVStore    | P2P for KVStore chain                | No      |
| 36658 | TCP      | ABCI KVStore    | ABCI socket (internal only)          | No      |

---

## 6. Local Development Setup

### Building MyChain Locally

```bash
cd mychain

# Install wasmvm dependency (Linux)
# The wasmvm C library must be available for CGO
# On Docker this is handled automatically; locally you need it:
wget https://github.com/CosmWasm/wasmvm/releases/download/v2.1.4/libwasmvm_muslc.x86_64.a \
  -O /usr/local/lib/libwasmvm_muslc.x86_64.a

# Build
make build        # -> ./build/mychaind

# Or install globally
make install      # -> $GOPATH/bin/mychaind

# Run tests
make test

# Lint
make lint
```

### Building ABCI KVStore Locally

```bash
cd abci-kvstore

# Build (no CGO needed)
go build -o kvstore-app .

# Run
./kvstore-app --addr tcp://0.0.0.0:36658 --db-path /tmp/kvstore-db
```

### Building API Backend Locally

```bash
cd api-backend

# Build (needs wasmvm for Cosmos SDK imports)
CGO_ENABLED=1 go build -o api-backend .

# Run (set env vars first)
export CHAIN_GRPC_ADDR=localhost:9090
export CHAIN_RPC_ADDR=http://localhost:26657
export KVSTORE_RPC_ADDR=http://localhost:36657
export CHAIN_ID=mychain-1
export TEST_MNEMONIC="abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
./api-backend
```

### Building the CosmWasm Contract Locally

```bash
cd contracts/task-bounty

# Add WASM target
rustup target add wasm32-unknown-unknown

# Build
cargo build --release --target wasm32-unknown-unknown

# The WASM file is at:
# target/wasm32-unknown-unknown/release/task_bounty.wasm

# Run tests
cargo test

# Optimize (optional, reduces WASM size for production)
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.16.0
```

### Protobuf Code Generation

```bash
cd mychain

# Requires buf CLI and Docker
make proto-gen
```

This uses the `ghcr.io/cosmos/proto-builder:0.14.0` Docker image to generate Go code from `.proto` files.

---

## 7. Running All Components Locally

Open **4 terminal windows**:

```
Terminal 1: Initialize and start MyChain
+------------------------------------------+
| cd mychain                               |
| ./scripts/init.sh                        |
| # This initializes genesis and starts    |
| # the node on ports 26656/26657/1317/9090|
+------------------------------------------+

Terminal 2: Start CometBFT + KVStore
+------------------------------------------+
| # First init CometBFT for kvstore        |
| cometbft init --home /tmp/kvstore-cmt    |
| # Configure proxy_app in config.toml     |
| # Start kvstore app                      |
| cd abci-kvstore                          |
| ./kvstore-app                            |
| # Then start CometBFT                    |
| cometbft start --home /tmp/kvstore-cmt   |
+------------------------------------------+

Terminal 3: Start API Backend
+------------------------------------------+
| cd api-backend                           |
| export TEST_MNEMONIC="abandon ..."       |
| ./api-backend                            |
| # Listening on :8080                     |
+------------------------------------------+

Terminal 4: Test
+------------------------------------------+
| curl http://localhost:8080/api/health    |
+------------------------------------------+
```

---

## 8. Resetting State

### Full Reset (Docker)

```bash
# Stop everything and remove volumes
docker compose down -v

# Rebuild and start fresh
docker compose up --build -d
```

### Partial Reset (Keep Images)

```bash
# Stop and remove volumes only
docker compose down -v

# Start without rebuilding
docker compose up -d
```

### When You Need a Reset

- After changing genesis configuration (accounts, balances)
- After a consensus failure (corrupted state)
- To start with a clean chain (no tasks, bounties, or KV data)
- After changing protobuf definitions

---

## 9. Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f mychain-node
docker compose logs -f abci-kvstore
docker compose logs -f api-backend

# Last 100 lines
docker compose logs --tail 100 api-backend
```

### What to Look For

| Log Line Pattern                     | Meaning                          |
|--------------------------------------|----------------------------------|
| `committed state`                    | Block committed (chain working)  |
| `[api] POST /api/tasks`             | API request received             |
| `[chain-client] BroadcastTx`        | Transaction sent to chain        |
| `[kv-client] BroadcastTxCommit`     | KV write sent                    |
| `ERR` or `error`                    | Something went wrong             |
| `waiting for chain to be ready`     | API backend polling for startup  |
| `contract address resolved`         | WASM contract found              |

---

## 10. Verifying Block Production

```bash
# Check current block height via CometBFT RPC
curl http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Check KVStore block height
curl http://localhost:36657/status | jq '.result.sync_info.latest_block_height'

# Both should be incrementing every ~5 seconds
```

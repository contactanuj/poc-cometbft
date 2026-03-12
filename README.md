# CometBFT Full Stack POC

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

| Document | Description |
|----------|-------------|
| [01-OVERVIEW](docs/01-OVERVIEW.md) | Architecture, components, how it all fits together |
| [02-GLOSSARY](docs/02-GLOSSARY.md) | All blockchain/Cosmos/CosmWasm terms explained with analogies |
| [03-ARCHITECTURE](docs/03-ARCHITECTURE.md) | Deep architecture, data flow diagrams, design decisions |
| [04-SETUP-GUIDE](docs/04-SETUP-GUIDE.md) | Complete setup, install, build, and run instructions |
| [05-TASK-MODULE](docs/05-TASK-MODULE-GUIDE.md) | Cosmos SDK custom module guide with tutorials |
| [06-BOUNTY-CONTRACT](docs/06-BOUNTY-CONTRACT-GUIDE.md) | CosmWasm smart contract guide with payout math |
| [07-KVSTORE](docs/07-KVSTORE-GUIDE.md) | Direct ABCI application guide |
| [08-API-REFERENCE](docs/08-API-REFERENCE.md) | All 12 REST endpoints with full schemas |
| [09-PRODUCTION](docs/09-PRODUCTION-GUIDE.md) | Scaling, monitoring, security, Kubernetes deployment |
| [10-DEVELOPMENT](docs/10-DEVELOPMENT-GUIDE.md) | Extending the project (new modules, contracts, endpoints) |
| [11-TROUBLESHOOTING](docs/11-TROUBLESHOOTING.md) | Common issues and solutions |

---

A proof-of-concept demonstrating all three contract/logic approaches in the CometBFT/Cosmos ecosystem:

1. **Cosmos SDK Custom Module** (Go) — `x/taskreg` module for task management
2. **CosmWasm Smart Contract** (Rust) — `task-bounty` contract for bounty management
3. **Direct ABCI Application** (Go) — Simple key-value store with Badger DB

Unified by a REST API backend and Docker deployment.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    REST API Backend (:8080)                 │
│  POST /api/tasks    POST /api/bounties/fund   PUT /api/kv   │
└───────┬──────────────────────┬─────────────────────┬────────┘
        │ gRPC/RPC             │ gRPC/RPC            │ RPC
        ▼                      ▼                      ▼
┌───────────────┐    ┌─────────────────┐    ┌──────────────────┐
│  mychain-node │    │  mychain-node   │    │   abci-kvstore   │
│  (SDK Chain)  │    │  (CosmWasm)     │    │   (Direct ABCI)  │
│  :26657/9090  │    │  same node      │    │   :36657         │
│               │    │                 │    │                  │
│ x/taskreg mod │    │ task-bounty.wasm│    │ Badger v4 state  │
└───────────────┘    └─────────────────┘    └──────────────────┘
```

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local development)
- Rust 1.75+ (for contract development)
- `buf` CLI (for proto generation)

## Quick Start

```bash
# Start all services
docker compose up --build

# In another terminal, deploy the CosmWasm contract
# (requires mychaind binary or exec into the container)
docker exec mychain-node /scripts/deploy-contracts.sh
```

## API Reference

### Health Check
```bash
GET /api/health
```

### SDK Module — Task Management
```bash
# Create a task
POST /api/tasks
{"title": "My Task", "description": "Do something"}

# List all tasks
GET /api/tasks

# Get a task
GET /api/tasks/{id}

# Assign a task
POST /api/tasks/{id}/assign
{"assignee": "cosmos1..."}

# Complete a task
POST /api/tasks/{id}/complete
```

### CosmWasm — Bounty Management
```bash
# Fund a bounty
POST /api/bounties/fund
{"task_id": 1, "amount": "1000", "denom": "token"}

# Claim a bounty
POST /api/bounties/claim
{"task_id": 1}

# Get a bounty
GET /api/bounties/{id}

# List bounties
GET /api/bounties

# Get contract config
GET /api/config
```

### ABCI KV Store
```bash
# Set a key
PUT /api/kv/{key}
{"value": "hello"}

# Get a key
GET /api/kv/{key}
```

## Version Matrix

| Component | Version |
|-----------|---------|
| Cosmos SDK | v0.50.14 |
| CometBFT | v0.38.12 |
| wasmd | v0.53.4 |
| wasmvm | v2.1.4 |
| cosmwasm-std | 2.1 |
| Go | 1.22+ |
| Rust | 1.75+ |

## Project Structure

```
poc-cometbft/
├── mychain/           # Cosmos SDK chain with x/taskreg module + wasmd
├── contracts/         # CosmWasm smart contracts (Rust)
│   └── task-bounty/
├── abci-kvstore/      # Direct ABCI application with Badger DB
├── api-backend/       # REST API backend (chi router)
├── docker-compose.yml
├── deploy-contracts.sh
└── go.work
```

## Development

### Build locally
```bash
cd mychain && go build ./cmd/mychaind
cd contracts/task-bounty && cargo test
cd abci-kvstore && go build .
cd api-backend && go build .
```

### Generate protobuf
```bash
cd mychain && make proto-gen
```

## Production Note

Production Cosmos SDK chains use the built-in gRPC-Gateway for REST endpoints.
The separate API backend in this POC demonstrates programmatic interaction patterns
and serves as an educational example.

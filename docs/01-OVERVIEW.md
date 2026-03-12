# Project Overview: CometBFT Full-Stack POC

## Documentation Index

| # | Document | Purpose | Audience |
|---|----------|---------|----------|
| 01 | [Overview](01-OVERVIEW.md) | Architecture, components, how it fits together | Everyone |
| 02 | [Glossary](02-GLOSSARY.md) | All terminologies with technical + everyday explanations | Beginners |
| 03 | [Architecture](03-ARCHITECTURE.md) | Deep architecture, data flows, ASCII diagrams | Developers, Architects |
| 04 | [Setup Guide](04-SETUP-GUIDE.md) | Install, build, run -- Docker and local | Developers, DevOps |
| 05 | [Task Module Guide](05-TASK-MODULE-GUIDE.md) | Cosmos SDK custom module deep dive | Developers |
| 06 | [Bounty Contract Guide](06-BOUNTY-CONTRACT-GUIDE.md) | CosmWasm smart contract deep dive | Developers |
| 07 | [KVStore Guide](07-KVSTORE-GUIDE.md) | Direct ABCI application deep dive | Developers |
| 08 | [API Reference](08-API-REFERENCE.md) | All 12 REST endpoints with schemas | Developers |
| 09 | [Production Guide](09-PRODUCTION-GUIDE.md) | Scaling, monitoring, security, Kubernetes | DevOps, SRE |
| 10 | [Development Guide](10-DEVELOPMENT-GUIDE.md) | Extending the project, adding modules/contracts | Developers |
| 11 | [Troubleshooting](11-TROUBLESHOOTING.md) | Common issues and solutions | Everyone |

## Who Should Read What

| Role | Start Here | Then Read |
|------|-----------|-----------|
| **New to blockchain** | 02-GLOSSARY -> 01-OVERVIEW | 04-SETUP -> 05/06/07 (pick one) |
| **Developer (backend)** | 01-OVERVIEW -> 04-SETUP | 08-API-REF -> 05/06/07 -> 10-DEV |
| **Developer (blockchain)** | 03-ARCHITECTURE -> 05-TASK | 06-BOUNTY -> 07-KVSTORE -> 10-DEV |
| **DevOps / SRE** | 01-OVERVIEW -> 04-SETUP | 09-PRODUCTION -> 11-TROUBLESHOOTING |
| **Architect** | 03-ARCHITECTURE -> 09-PRODUCTION | 02-GLOSSARY (for terminology) |

---

## What This Project Is

This is a **proof-of-concept** demonstrating three different approaches to building blockchain applications using the **CometBFT** consensus engine (formerly Tendermint). It is a complete, working full-stack blockchain system with a REST API backend that unifies all three approaches under a single HTTP interface.

---

## System Architecture

```
                            +-------------------------------+
                            |        User / Client          |
                            |     (curl, Postman, App)      |
                            +---------------+---------------+
                                            |
                                       HTTP :8080
                                            |
                            +---------------v---------------+
                            |       REST API Backend        |
                            |       (Go + Chi Router)       |
                            |                               |
                            |  /api/tasks    -> gRPC :9090  |
                            |  /api/bounties -> gRPC + RPC  |
                            |  /api/kv       -> RPC  :36657 |
                            +-----+------------+-------+----+
                                  |            |       |
                     gRPC :9090   |            |       |  RPC :36657
                     RPC  :26657  |            |       |
                                  |            |       |
              +-------------------v--+         |   +---v-------------------+
              |   MyChain Node       |         |   |  ABCI KVStore         |
              |   (Cosmos SDK)       |         |   |  (Direct ABCI)        |
              |                      |         |   |                       |
              | +------------------+ |         |   | +-------------------+ |
              | | x/taskreg module | |         |   | | KVStoreApp (Go)   | |
              | | (Task Registry)  | |         |   | | Badger DB storage | |
              | +------------------+ |         |   | +-------------------+ |
              |                      |         |   |                       |
              | +------------------+ |         |   | +-------------------+ |
              | | x/wasm (wasmd)   | |         |   | | CometBFT          | |
              | | CosmWasm Runtime |<----+     |   | | (separate instance)| |
              | +------------------+ |   |     |   | +-------------------+ |
              |                      |   |     |   |                       |
              | +------------------+ |   |     |   | Ports:               |
              | | CometBFT         | |   |     |   |  36656 P2P           |
              | | (embedded)       | |   |     |   |  36657 RPC           |
              | +------------------+ |   |     |   |  36658 ABCI          |
              |                      |   |     |   +-----------------------+
              | Ports:               |   |     |
              |  26656 P2P           |   |     |
              |  26657 RPC           |   |     |
              |  1317  REST API      |   |     |
              |  9090  gRPC          |   |     |
              +----------------------+   |     |
                                         |     |
                        +----------------+-----+
                        |  Task Bounty Contract |
                        |  (Rust -> WASM)       |
                        |  Runs INSIDE MyChain  |
                        |  via wasmd module     |
                        +-----------------------+
```

---

## The Three Approaches

### Approach 1: Cosmos SDK Custom Module -- Task Registry

- **Language**: Go
- **What it does**: A native blockchain module for creating, assigning, and completing tasks
- **When to use in production**: Maximum performance, deep chain integration, custom consensus-level logic
- **Analogy**: Building a feature directly into the operating system kernel

### Approach 2: CosmWasm Smart Contract -- Task Bounty

- **Language**: Rust (compiled to WebAssembly)
- **What it does**: A smart contract for managing financial bounties on tasks
- **When to use in production**: Upgradable business logic, sandboxed execution, no chain restart needed
- **Analogy**: Installing an application on the operating system

### Approach 3: Direct ABCI Application -- Key-Value Store

- **Language**: Go
- **What it does**: A bare-bones key-value store built directly against CometBFT's ABCI interface
- **When to use in production**: Full control, minimal overhead, custom consensus rules
- **Analogy**: Writing your own device driver that talks directly to hardware

---

## Component Map

| Component        | Directory         | Language | Port(s)                                          | Purpose                          |
|------------------|-------------------|----------|--------------------------------------------------|----------------------------------|
| MyChain Node     | `mychain/`        | Go       | 26657 (RPC), 9090 (gRPC), 1317 (REST), 26656 (P2P) | Cosmos SDK blockchain node       |
| Task Bounty      | `contracts/`      | Rust     | Runs inside MyChain                              | CosmWasm smart contract          |
| ABCI KVStore     | `abci-kvstore/`   | Go       | 36657 (RPC), 36658 (ABCI), 36656 (P2P)          | Standalone ABCI application      |
| API Backend      | `api-backend/`    | Go       | 8080                                             | Unified REST API for all three   |

---

## Version Matrix

| Technology     | Version   | Role                              |
|----------------|-----------|-----------------------------------|
| Go             | 1.22      | Chain, API backend, KVStore       |
| Rust           | 1.75+     | Smart contract                    |
| Cosmos SDK     | 0.50.14   | Blockchain framework              |
| CometBFT       | 0.38.12   | Consensus engine                  |
| wasmd          | 0.53.4    | CosmWasm module for Cosmos SDK    |
| wasmvm         | 2.1.4     | WASM virtual machine (C library)  |
| cosmwasm-std   | 2.1       | Rust smart contract standard lib  |
| IBC-Go         | 8.4.0     | Inter-Blockchain Communication    |
| Badger DB      | 4.2.0     | KVStore persistence               |
| Chi Router     | 5.1.0     | HTTP routing for API backend      |

---

## How a Request Flows Through the System

Here is what happens when a user creates a task (simplified):

```
1. User sends:           POST http://localhost:8080/api/tasks
                          {"title": "Fix bug", "description": "Fix login"}

2. API Backend receives:  Chi router matches /api/tasks -> TaskHandler.CreateTask()

3. Handler builds Msg:    MsgCreateTask{Creator: "cosmos1...", Title: "Fix bug", ...}

4. Client signs Tx:       Gets account sequence via gRPC -> Signs with private key

5. Broadcasts to chain:   Sends signed bytes to CometBFT RPC (:26657)

6. CometBFT validates:    CheckTx -> ante handler chain (sig verify, gas, etc.)

7. CometBFT consensus:    Block proposed -> Validators vote -> 2/3+ agree

8. Chain executes:         FinalizeBlock -> msg_server.CreateTask() -> keeper.SetTask()

9. State committed:        Commit() -> IAVL tree persisted to disk

10. Response returned:     API Backend returns {"success": true, "data": {"tx_hash": "A1B2..."}}
```

---

## Quick Start

```bash
# Clone and start everything
git clone <repo-url>
cd poc-cometbft
docker compose up --build -d

# Wait ~30 seconds for chain to start producing blocks

# Verify health
curl http://localhost:8080/api/health

# Create your first task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "My first blockchain task"}'
```

See [04-SETUP-GUIDE.md](04-SETUP-GUIDE.md) for complete instructions.

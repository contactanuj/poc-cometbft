# Development Guide

How to extend the project: add new modules, contracts, API endpoints, and more.

---

## 1. Project Structure

```
poc-cometbft/
|
+-- go.work                        # Go workspace (links mychain + api-backend)
+-- Cargo.toml                     # Rust workspace (links contracts)
+-- docker-compose.yml             # Container orchestration
+-- deploy-contracts.sh            # Contract deployment script
|
+-- mychain/                       # Cosmos SDK blockchain
|   +-- Makefile                   # Build targets
|   +-- Dockerfile                 # Container build
|   +-- go.mod                     # Go dependencies
|   +-- app/
|   |   +-- app.go                 # Application definition (23 modules)
|   |   +-- ante.go                # Transaction validation chain
|   |   +-- encoding.go            # Codec configuration
|   |   +-- export.go              # State export
|   +-- cmd/mychaind/
|   |   +-- main.go                # CLI entry point
|   |   +-- root.go                # Root command and subcommands
|   +-- proto/mychain/taskreg/v1/  # Protobuf definitions
|   |   +-- types.proto            # Data structures
|   |   +-- tx.proto               # Messages
|   |   +-- query.proto            # Queries
|   |   +-- genesis.proto          # Genesis state
|   +-- scripts/
|   |   +-- init.sh                # Chain initialization
|   +-- x/taskreg/                 # Custom module
|       +-- module.go              # Module registration
|       +-- keeper/                # State access layer
|       +-- types/                 # Types, errors, events
|
+-- contracts/task-bounty/         # CosmWasm smart contract
|   +-- Cargo.toml
|   +-- src/
|       +-- lib.rs                 # Module exports
|       +-- contract.rs            # Handlers + tests
|       +-- msg.rs                 # Message definitions
|       +-- state.rs               # State structures
|       +-- error.rs               # Custom errors
|
+-- abci-kvstore/                  # Direct ABCI application
|   +-- main.go                    # Entry point
|   +-- app.go                     # ABCI implementation
|   +-- state.go                   # Badger DB wrapper
|   +-- Dockerfile
|   +-- entrypoint.sh
|
+-- api-backend/                   # REST API server
    +-- main.go                    # Router setup
    +-- Dockerfile
    +-- go.mod
    +-- client/
    |   +-- chain.go               # Chain client (gRPC + RPC)
    |   +-- kvstore.go             # KVStore client (RPC)
    +-- handlers/
        +-- health.go              # Health check
        +-- tasks.go               # Task endpoints
        +-- bounties.go            # Bounty endpoints
        +-- kvstore.go             # KV endpoints
        +-- util.go                # Response helpers
```

### Dependency Graph

```
go.work
  |
  +-- mychain (independent, builds alone)
  |
  +-- api-backend (depends on mychain for types + encoding)

Cargo workspace
  |
  +-- contracts/task-bounty (independent Rust crate)
```

---

## 2. Adding a New Cosmos SDK Module

Follow the `x/taskreg` module as a template. Here is the step-by-step process for a hypothetical `x/reward` module:

### Step 1: Define Protobuf Messages

Create `proto/mychain/reward/v1/`:

```protobuf
// types.proto
syntax = "proto3";
package mychain.reward.v1;

message Reward {
  uint64 id = 1;
  string recipient = 2;
  string amount = 3;
  string denom = 4;
  bool distributed = 5;
}

// tx.proto
message MsgCreateReward {
  string creator = 1;
  string recipient = 2;
  string amount = 3;
  string denom = 4;
}
message MsgCreateRewardResponse {
  uint64 id = 1;
}

// query.proto
message QueryRewardRequest {
  uint64 id = 1;
}
message QueryRewardResponse {
  Reward reward = 1;
}

// genesis.proto
message GenesisState {
  repeated Reward rewards = 1;
  uint64 next_reward_id = 2;
}
```

### Step 2: Generate Go Code

```bash
cd mychain
make proto-gen
```

This produces `x/reward/types/*.pb.go` files.

### Step 3: Create Module Structure

```
x/reward/
  +-- module.go            # AppModuleBasic + AppModule interfaces
  +-- keeper/
  |   +-- keeper.go        # Keeper struct with SetReward, GetReward
  |   +-- msg_server.go    # MsgCreateReward handler
  |   +-- grpc_query.go    # QueryReward handler
  |   +-- genesis.go       # InitGenesis, ExportGenesis
  +-- types/
      +-- keys.go          # Store key prefix: "Reward/value/"
      +-- errors.go        # ErrRewardNotFound, etc.
      +-- events.go        # EventTypeCreateReward
      +-- codec.go         # RegisterInterfaces, RegisterLegacyAminoCodec
      +-- msgs.go          # ValidateBasic, GetSigners for messages
      +-- expected_keepers.go  # Interfaces for dependent keepers
      +-- genesis.go       # DefaultGenesis, Validate
```

**Tip**: Copy `x/taskreg` and rename. Most of the boilerplate is identical.

### Step 4: Register in app.go

```go
// In app/app.go, add:

// 1. Import
import rewardkeeper "poc-cometbft/mychain/x/reward/keeper"
import rewardtypes "poc-cometbft/mychain/x/reward/types"

// 2. Add keeper to MyChainApp struct
type MyChainApp struct {
    // ...existing keepers...
    RewardKeeper rewardkeeper.Keeper
}

// 3. Add store key
keys := storetypes.NewKVStoreKeys(
    // ...existing keys...
    rewardtypes.StoreKey,
)

// 4. Initialize keeper
app.RewardKeeper = rewardkeeper.NewKeeper(
    appCodec,
    keys[rewardtypes.StoreKey],
    app.BankKeeper,
)

// 5. Add to module manager
app.ModuleManager = module.NewManager(
    // ...existing modules...
    reward.NewAppModule(appCodec, app.RewardKeeper),
)

// 6. Add to BeginBlockers, EndBlockers, InitGenesis ordering
```

### Step 5: Add API Endpoint

In `api-backend/handlers/`:

```go
// rewards.go
type RewardHandler struct {
    client *client.ChainClient
}

func (h *RewardHandler) CreateReward(w http.ResponseWriter, r *http.Request) {
    // Parse request, build message, broadcast
}
```

Wire in `api-backend/main.go`:

```go
rewardHandler := handlers.NewRewardHandler(chainClient)
r.Post("/api/rewards", rewardHandler.CreateReward)
r.Get("/api/rewards/{id}", rewardHandler.GetReward)
```

---

## 3. Adding a New CosmWasm Contract

### Step 1: Create Contract Scaffold

```bash
cd contracts

# Option A: Manual (copy task-bounty and modify)
cp -r task-bounty my-new-contract

# Option B: Use cargo-generate (if installed)
cargo generate --git https://github.com/CosmWasm/cw-template.git --name my-new-contract
```

### Step 2: Define Messages

Edit `src/msg.rs`:

```rust
#[cw_serde]
pub struct InstantiateMsg {
    pub owner: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    DoSomething { param: String },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(ConfigResponse)]
    Config {},
}
```

### Step 3: Implement Handlers

Edit `src/contract.rs`:

```rust
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(deps: DepsMut, _env: Env, info: MessageInfo, msg: InstantiateMsg)
    -> Result<Response, ContractError> {
    // Save config, validate inputs
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg)
    -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::DoSomething { param } => execute_do_something(deps, info, param),
    }
}
```

### Step 4: Add to Workspace

Edit root `Cargo.toml`:

```toml
[workspace]
members = [
    "contracts/task-bounty",
    "contracts/my-new-contract",   # Add this
]
```

### Step 5: Build and Deploy

```bash
# Build
cd contracts/my-new-contract
cargo build --release --target wasm32-unknown-unknown

# Test
cargo test

# Deploy (after docker compose is running)
docker compose exec mychain-node mychaind tx wasm store /path/to/my_new_contract.wasm \
  --from admin --gas auto --gas-adjustment 1.3 -y
```

---

## 4. Adding a New API Endpoint

### Step 1: Create Handler

Create `api-backend/handlers/my_handler.go`:

```go
package handlers

import (
    "net/http"
    "poc-cometbft/api-backend/client"
)

type MyHandler struct {
    client *client.ChainClient
}

func NewMyHandler(c *client.ChainClient) *MyHandler {
    return &MyHandler{client: c}
}

func (h *MyHandler) DoSomething(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req struct {
        Param string `json:"param"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // 2. Call chain client
    result, err := h.client.DoSomething(req.Param)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // 3. Return response
    respondJSON(w, http.StatusOK, result)
}
```

### Step 2: Wire in Router

Edit `api-backend/main.go`:

```go
myHandler := handlers.NewMyHandler(chainClient)
r.Post("/api/my-endpoint", myHandler.DoSomething)
r.Get("/api/my-endpoint/{id}", myHandler.GetById)
```

### Step 3: Add Client Method (if needed)

Edit `api-backend/client/chain.go`:

```go
func (c *ChainClient) DoSomething(param string) (string, error) {
    msg := &sometypes.MsgDoSomething{
        Creator: c.signerAddress,
        Param:   param,
    }
    return c.broadcastTx(msg)
}
```

---

## 5. Protobuf Workflow

### File Structure

```
proto/
  +-- buf.yaml              # Linting and dependency config
  +-- buf.gen.gogo.yaml     # Code generation template
  +-- mychain/
      +-- taskreg/v1/       # Module proto files
          +-- types.proto
          +-- tx.proto
          +-- query.proto
          +-- genesis.proto
```

### Generation Command

```bash
cd mychain
make proto-gen
```

This runs:

```bash
docker run --rm -v $(pwd):/workspace --workdir /workspace \
  ghcr.io/cosmos/proto-builder:0.14.0 \
  buf generate --template proto/buf.gen.gogo.yaml proto/
```

### Generated Files

Proto files generate `.pb.go` files in the `types/` directory:

```
x/taskreg/types/
  +-- types.pb.go       # From types.proto (Task, TaskStatus)
  +-- tx.pb.go          # From tx.proto (MsgCreateTask, etc.)
  +-- query.pb.go       # From query.proto (QueryTaskRequest, etc.)
  +-- genesis.pb.go     # From genesis.proto (GenesisState)
```

**Do not edit generated files.** Always modify the `.proto` source and regenerate.

---

## 6. Testing

### Go Tests (MyChain)

```bash
cd mychain
make test          # Run all tests
go test ./x/taskreg/...  # Run module tests only
```

### Go Tests (API Backend)

```bash
cd api-backend
go test ./...
```

### Rust Contract Tests

```bash
cd contracts/task-bounty
cargo test                # Run all tests
cargo test test_claim     # Run specific test
cargo test -- --nocapture # Show println! output
```

The contract tests use `cw-multi-test` which simulates the blockchain environment:

```rust
#[test]
fn test_fund_bounty() {
    let mut app = App::default();
    // ... setup contract, send messages, assert state
}
```

### Integration Testing (Docker)

```bash
# Start stack
docker compose up --build -d

# Wait for health
until curl -s http://localhost:8080/api/health | grep -q '"success":true'; do sleep 2; done

# Run test script
bash test_integration.sh

# Example test assertions
RESULT=$(curl -s http://localhost:8080/api/tasks)
echo "$RESULT" | jq -e '.success == true' || echo "FAIL: tasks list"
```

---

## 7. Common Development Tasks

### Rebuild After Code Changes

```bash
# Rebuild everything
docker compose up --build -d

# Rebuild only one service
docker compose up --build -d api-backend

# Rebuild with no cache (clean build)
docker compose build --no-cache
```

### Reset Chain State

```bash
docker compose down -v     # Remove all data
docker compose up -d       # Start fresh
```

### Debug Transaction Failures

```bash
# 1. Check API backend logs
docker compose logs -f api-backend

# 2. Check chain logs
docker compose logs -f mychain-node

# 3. Query transaction by hash
curl "http://localhost:26657/tx?hash=0xABCD..." | jq

# 4. Check account sequence
curl http://localhost:1317/cosmos/auth/v1beta1/accounts/cosmos1... | jq
```

### Inspect On-Chain State Directly

```bash
# Enter the mychain container
docker compose exec mychain-node bash

# Query tasks via CLI
mychaind query taskreg list-tasks --output json

# Query a specific task
mychaind query taskreg task 1 --output json

# Query account balances
mychaind query bank balances cosmos1... --output json

# Query WASM contract state
mychaind query wasm contract-state smart $CONTRACT_ADDR '{"config":{}}' --output json

# List stored WASM codes
mychaind query wasm list-code --output json
```

### Watch Block Production

```bash
# Stream new blocks
curl -s "http://localhost:26657/subscribe?query=\"tm.event='NewBlock'\"" | jq
```

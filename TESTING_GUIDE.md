# Testing Guide — CometBFT Full Stack POC

This guide walks through testing all three approaches (SDK module, CosmWasm contract, direct ABCI app) via the unified REST API backend.

## Prerequisites

- Docker & Docker Compose installed
- `curl` (or any HTTP client like Postman/Insomnia)
- Ports 8080, 26657, 36657, 9090, 1317 available

## 1. Start the Stack

```bash
cd poc-cometbft

# Build and start all 3 containers
docker compose up --build -d
```

This starts:

| Container | Description | Ports |
|-----------|-------------|-------|
| `mychain-node` | Cosmos SDK chain (x/taskreg + CosmWasm) | 26657, 9090, 1317 |
| `abci-kvstore` | Direct ABCI key-value store (Badger DB) | 36657 |
| `api-backend` | REST API gateway | 8080 |

Wait ~60 seconds for all containers to become healthy. You can monitor with:

```bash
docker compose ps
```

All three should show `healthy` or `running` status. You can also watch the logs:

```bash
docker compose logs -f
```

## 2. Health Check

Verify all services are connected:

```bash
curl -s http://localhost:8080/api/health | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "chain": "ok",
    "kvstore": "ok"
  }
}
```

You can also verify the chain is producing blocks:

```bash
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

## 3. SDK Module — Task Lifecycle (x/taskreg)

This tests the custom Cosmos SDK module. Tasks go through the lifecycle: OPEN (1) -> ASSIGNED (2) -> COMPLETED (3).

### 3.1 Create a Task

```bash
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Fix login bug", "description": "Users cannot log in on mobile"}' | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "tx_hash": "B910C22DB03A3CE3F5743B0D7209AD82A02644FBF6165805C947B10EE56252DC"
  }
}
```

The `tx_hash` is the blockchain transaction hash. Wait ~3 seconds for the block to be committed.

### 3.2 List All Tasks

```bash
curl -s http://localhost:8080/api/tasks | jq
```

Expected:

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "title": "Fix login bug",
      "description": "Users cannot log in on mobile",
      "creator": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4",
      "status": 1
    }
  ]
}
```

Status `1` = OPEN. The `creator` is the admin account derived from the test mnemonic.

### 3.3 Get a Single Task

```bash
curl -s http://localhost:8080/api/tasks/1 | jq
```

### 3.4 Create a Second Task

```bash
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Add dark mode", "description": "Support dark theme across the app"}' | jq
```

Wait ~3 seconds, then list again to see both tasks:

```bash
curl -s http://localhost:8080/api/tasks | jq
```

### 3.5 Assign a Task

Assign task 1 to the same admin address (since we only have one signing key in the API backend):

```bash
curl -s -X POST http://localhost:8080/api/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d '{"assignee": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4"}' | jq
```

Wait ~3 seconds, then verify the status changed to ASSIGNED (2):

```bash
curl -s http://localhost:8080/api/tasks/1 | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "id": 1,
    "title": "Fix login bug",
    "description": "Users cannot log in on mobile",
    "creator": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4",
    "assignee": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4",
    "status": 2
  }
}
```

### 3.6 Complete a Task

```bash
curl -s -X POST http://localhost:8080/api/tasks/1/complete | jq
```

Wait ~3 seconds, then verify status is COMPLETED (3):

```bash
curl -s http://localhost:8080/api/tasks/1 | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "id": 1,
    "title": "Fix login bug",
    "description": "Users cannot log in on mobile",
    "creator": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4",
    "assignee": "cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4",
    "status": 3
  }
}
```

### 3.7 Error Cases

Try completing a task that is already completed:

```bash
curl -s -X POST http://localhost:8080/api/tasks/1/complete | jq
```

Try getting a task that doesn't exist:

```bash
curl -s http://localhost:8080/api/tasks/999 | jq
```

Try creating a task without a title:

```bash
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"description": "no title"}' | jq
```

## 4. ABCI KV Store — Direct CometBFT Application

This tests the standalone ABCI application that uses Badger DB for state, with CometBFT for consensus. Transactions are in `key=value` format broadcast directly to CometBFT.

### 4.1 Store a Value

```bash
curl -s -X PUT http://localhost:8080/api/kv/greeting \
  -H "Content-Type: application/json" \
  -d '{"value": "hello world"}' | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "key": "greeting",
    "tx_hash": "6823fa03fb464f179bac63c6c9240d4962a65655b28de138dab8cffc41a2fe20",
    "value": "hello world"
  }
}
```

### 4.2 Retrieve a Value

```bash
curl -s http://localhost:8080/api/kv/greeting | jq
```

Expected:

```json
{
  "success": true,
  "data": {
    "key": "greeting",
    "value": "hello world"
  }
}
```

### 4.3 Update a Value

```bash
curl -s -X PUT http://localhost:8080/api/kv/greeting \
  -H "Content-Type: application/json" \
  -d '{"value": "hello cosmos"}' | jq
```

Wait ~3 seconds and read again:

```bash
curl -s http://localhost:8080/api/kv/greeting | jq
```

The value should now be `"hello cosmos"`.

### 4.4 Store Multiple Keys

```bash
curl -s -X PUT http://localhost:8080/api/kv/name \
  -H "Content-Type: application/json" \
  -d '{"value": "Alice"}' | jq

curl -s -X PUT http://localhost:8080/api/kv/counter \
  -H "Content-Type: application/json" \
  -d '{"value": "42"}' | jq
```

Wait ~3 seconds and retrieve:

```bash
curl -s http://localhost:8080/api/kv/name | jq
curl -s http://localhost:8080/api/kv/counter | jq
```

### 4.5 Read a Non-Existent Key

```bash
curl -s http://localhost:8080/api/kv/nonexistent | jq
```

Returns an empty value.

## 5. CosmWasm — Bounty Contract (Optional)

The bounty contract endpoints require deploying the CosmWasm contract first. If no contract is deployed, these endpoints will return errors.

To deploy the contract, you need to:

1. Build the contract: `cd contracts/task-bounty && cargo build --release --target wasm32-unknown-unknown`
2. Optimize: use `cosmwasm/optimizer:0.16.0`
3. Store and instantiate on chain via `mychaind tx wasm store` and `mychaind tx wasm instantiate`

Once deployed, these endpoints become available:

```bash
# Fund a bounty for task 1
curl -s -X POST http://localhost:8080/api/bounties/fund \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1, "amount": "1000", "denom": "token"}' | jq

# Get bounty details
curl -s http://localhost:8080/api/bounties/1 | jq

# List all bounties
curl -s http://localhost:8080/api/bounties | jq

# Claim a bounty
curl -s -X POST http://localhost:8080/api/bounties/claim \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1}' | jq

# Get contract config
curl -s http://localhost:8080/api/config | jq
```

## 6. Observing Logs

Watch what happens behind the scenes in each container:

```bash
# All containers
docker compose logs -f

# Just the API backend (shows request routing and tx broadcasting)
docker compose logs -f api-backend

# Just the chain node (shows block production and tx execution)
docker compose logs -f mychain-node

# Just the ABCI kvstore (shows ABCI method calls)
docker compose logs -f abci-kvstore
```

Look for log prefixes to trace requests:
- `[api]` — HTTP handler in api-backend
- `[chain-client]` — gRPC/RPC calls to the chain
- `[kvstore-client]` — RPC calls to the ABCI kvstore
- `[taskreg]` — SDK module keeper operations on-chain

## 7. Direct Chain Access

You can also query the chain directly, bypassing the API backend:

### CometBFT RPC (mychain)

```bash
# Node status
curl -s http://localhost:26657/status | jq '.result.sync_info'

# Latest block
curl -s http://localhost:26657/block | jq '.result.block.header.height'

# Search for transactions
curl -s "http://localhost:26657/tx_search?query=\"tx.height>0\"" | jq '.result.total_count'
```

### CometBFT RPC (kvstore)

```bash
# Node status
curl -s http://localhost:36657/status | jq '.result.sync_info'

# Query a key directly via ABCI
curl -s "http://localhost:36657/abci_query?data=0x$(echo -n greeting | xxd -p)" | jq
```

### Cosmos REST API

```bash
# Account info
curl -s http://localhost:1317/cosmos/auth/v1beta1/accounts | jq

# Bank balances
curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos19rl4cm2hmr8afy4kldpxz3fka4jguq0auqdal4 | jq
```

## 8. Full Automated Test Script

Run this script to test the entire flow in one go:

```bash
#!/bin/bash
set -e

API="http://localhost:8080"

echo "=== Health Check ==="
curl -s $API/api/health | jq

echo -e "\n=== Create Task 1 ==="
curl -s -X POST $API/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Build dashboard","description":"Create admin dashboard"}' | jq
sleep 3

echo -e "\n=== Create Task 2 ==="
curl -s -X POST $API/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Write tests","description":"Add unit test coverage"}' | jq
sleep 3

echo -e "\n=== List All Tasks ==="
curl -s $API/api/tasks | jq

echo -e "\n=== Get Task 1 ==="
curl -s $API/api/tasks/1 | jq

CREATOR=$(curl -s $API/api/tasks/1 | jq -r '.data.creator')
echo "Creator address: $CREATOR"

echo -e "\n=== Assign Task 1 ==="
curl -s -X POST $API/api/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d "{\"assignee\":\"$CREATOR\"}" | jq
sleep 3

echo -e "\n=== Task 1 (should be ASSIGNED=2) ==="
curl -s $API/api/tasks/1 | jq '.data.status'

echo -e "\n=== Complete Task 1 ==="
curl -s -X POST $API/api/tasks/1/complete | jq
sleep 3

echo -e "\n=== Task 1 (should be COMPLETED=3) ==="
curl -s $API/api/tasks/1 | jq '.data.status'

echo -e "\n=== KV Store: Set ==="
curl -s -X PUT $API/api/kv/project \
  -H "Content-Type: application/json" \
  -d '{"value":"cometbft-poc"}' | jq
sleep 2

echo -e "\n=== KV Store: Get ==="
curl -s $API/api/kv/project | jq

echo -e "\n=== All Tests Passed ==="
```

Save as `test.sh` and run:

```bash
chmod +x test.sh
./test.sh
```

## 9. Cleanup

```bash
# Stop all containers
docker compose down

# Stop and remove all data (clean slate)
docker compose down -v
```

## Task Status Reference

| Code | Name | Description |
|------|------|-------------|
| 0 | UNSPECIFIED | Default zero value |
| 1 | OPEN | Task created, not yet assigned |
| 2 | ASSIGNED | Task assigned to someone |
| 3 | COMPLETED | Task marked as done |

## Troubleshooting

**Containers not starting?**
```bash
docker compose ps
docker compose logs mychain-node
```

**API returns connection refused?**
Wait for all healthchecks to pass (~60s after `docker compose up`).

**Task not found after creating?**
Transactions need ~3 seconds to be included in a block. Wait and retry.

**Sequence mismatch error?**
This happens when requests are sent too fast. Wait a few seconds between write operations (create/assign/complete). The API backend uses a mutex but the sequence can lag behind if blocks haven't been committed.

**Chain consensus failure in logs?**
Run `docker compose down -v` for a clean restart. This clears all chain state.

# Troubleshooting Guide

Common issues and their solutions, organized by category.

---

## 1. Build Issues

### wasmvm library not found

**Symptom:**
```
/usr/bin/ld: cannot find -lwasmvm_muslc
```

**Cause:** The wasmvm C library is required for CGO compilation of any code that imports Cosmos SDK WASM packages (including the API backend).

**Solution:**
```bash
# Linux (x86_64)
wget https://github.com/CosmWasm/wasmvm/releases/download/v2.1.4/libwasmvm_muslc.x86_64.a \
  -O /usr/local/lib/libwasmvm_muslc.x86_64.a

# macOS (via brew)
brew install wasmvm

# Docker: Handled automatically in Dockerfiles
```

---

### Go workspace module resolution errors

**Symptom:**
```
go: cannot find module for path poc-cometbft/mychain
```

**Cause:** The `go.work` file links `mychain` and `api-backend`. Building outside the workspace context fails.

**Solution:**
```bash
# Always build from the project root (where go.work is)
cd poc-cometbft
go build ./api-backend/...

# Or build inside the specific directory (uses go.work)
cd api-backend
go build .
```

---

### Rust wasm32 target not installed

**Symptom:**
```
error[E0463]: can't find crate for `std` (target: wasm32-unknown-unknown)
```

**Solution:**
```bash
rustup target add wasm32-unknown-unknown
```

---

### Docker build fails with memory error

**Symptom:**
```
signal: killed
# or
out of memory
```

**Cause:** Go compilation (especially with CGO) is memory-intensive. Docker Desktop may have insufficient memory.

**Solution:**
- Docker Desktop: Settings -> Resources -> Memory -> Set to 4GB+
- Linux: Check `docker info | grep Memory`

---

## 2. Startup Issues

### Container keeps restarting

**Diagnosis:**
```bash
docker compose ps          # Check status
docker compose logs mychain-node  # Check logs
```

**Common causes:**

| Log Message                          | Fix                                      |
|--------------------------------------|------------------------------------------|
| `panic: failed to initialize database` | `docker compose down -v` (corrupt state) |
| `address already in use`             | Stop conflicting processes on the port    |
| `genesis.json file already exists`   | `docker compose down -v` (reinitialize)  |

---

### API backend: "waiting for chain to be ready"

**Symptom:** API backend logs show repeated polling:

```
[api] waiting for chain to be ready... attempt 1/60
[api] waiting for chain to be ready... attempt 2/60
```

**Cause:** MyChain node takes 15-30 seconds to start. The API backend polls until it responds.

**Solution:** Wait. If it exceeds 60 attempts:
```bash
# Check if mychain is running
docker compose logs mychain-node | tail -20

# Common issue: mychain crashed
docker compose restart mychain-node
```

---

### Consensus failure after state change

**Symptom:**
```
ERR CONSENSUS FAILURE: appHash mismatch
```

**Cause:** State was modified outside of consensus (manual DB edits, version mismatch after rebuild).

**Solution:**
```bash
docker compose down -v
docker compose up --build -d
```

---

### Port already in use

**Symptom:**
```
bind: address already in use
```

**Solution:**
```bash
# Find what is using the port (e.g., 26657)
# Linux/Mac:
lsof -i :26657
# Windows:
netstat -ano | findstr 26657

# Kill the process or change the port in docker-compose.yml
```

---

## 3. Runtime Issues

### "account sequence mismatch"

**Symptom:**
```json
{"success": false, "error": "account sequence mismatch, expected 15, got 14"}
```

**Cause:** Two transactions were signed with the same sequence number. Usually happens when sending requests too fast.

**Solution:**
- Wait 1-2 seconds between write requests
- The mutex in the API backend prevents this in normal operation
- If persistent: restart the API backend (`docker compose restart api-backend`)

---

### Task not found immediately after creation

**Symptom:**
```bash
# Create task
curl -X POST http://localhost:8080/api/tasks -d '{"title":"test"}'
# Immediately query
curl http://localhost:8080/api/tasks/1
# -> error: task not found
```

**Cause:** The task creation transaction was submitted but not yet included in a block. Block time is ~5 seconds.

**Solution:** Wait 5 seconds (one block) before querying:
```bash
curl -X POST http://localhost:8080/api/tasks -d '{"title":"test"}'
sleep 5
curl http://localhost:8080/api/tasks/1  # Now it works
```

---

### Bounty endpoints return 500 with "contract not found"

**Symptom:**
```json
{"success": false, "error": "contract address not resolved"}
```

**Cause:** The CosmWasm contract was not deployed.

**Solution:**
```bash
docker compose exec mychain-node bash /scripts/deploy-contracts.sh
# Wait 10 seconds for deployment to complete
sleep 10
# Restart API backend to pick up the contract address
docker compose restart api-backend
```

---

### Connection refused

**Symptom:**
```json
{"success": false, "error": "connection refused"}
```

**Cause:** The target service is not running or not yet ready.

**Solution:**
```bash
# Check all services
docker compose ps

# Restart unhealthy services
docker compose restart

# Full restart
docker compose down
docker compose up -d
```

---

### KV store write timeout

**Symptom:** PUT /api/kv/key hangs for 30 seconds then times out.

**Cause:** The KVStore uses `BroadcastTxCommit` which waits for the block. If CometBFT is stuck, this hangs.

**Solution:**
```bash
# Check KVStore health
curl http://localhost:36657/status | jq '.result.sync_info.latest_block_height'

# If block height is not increasing:
docker compose restart abci-kvstore
```

---

## 4. Docker Issues

### Volumes not clearing on restart

**Symptom:** Old state persists after `docker compose down`.

**Cause:** `docker compose down` does not remove volumes by default.

**Solution:**
```bash
docker compose down -v   # -v removes named volumes
```

---

### Network connectivity between containers

**Symptom:** API backend cannot reach mychain-node.

**Diagnosis:**
```bash
# Check network
docker network ls | grep cometbft

# Test connectivity from api-backend container
docker compose exec api-backend curl http://mychain-node:26657/status
```

**Solution:**
```bash
# Recreate the network
docker compose down
docker compose up -d
```

---

### Disk space full

**Symptom:** Containers crash with I/O errors.

**Solution:**
```bash
# Check Docker disk usage
docker system df

# Clean up unused images, containers, volumes
docker system prune -a --volumes

# Clean up just this project
docker compose down -v
```

---

## 5. Log Reading Guide

### Log Prefixes

| Prefix             | Source                    | Meaning                           |
|--------------------|---------------------------|-----------------------------------|
| `[api]`            | API backend handlers      | HTTP request handling             |
| `[chain-client]`   | API backend ChainClient   | gRPC/RPC communication with chain |
| `[kv-client]`      | API backend KVStoreClient | RPC communication with KVStore    |
| `INF`              | CometBFT                  | Informational (normal operation)  |
| `ERR`              | CometBFT                  | Error (needs attention)           |
| `committed state`  | CometBFT                  | Block committed (chain healthy)   |

### Tracing a Request

To trace a task creation from end to end:

```bash
# 1. API backend receives request
[api] POST /api/tasks: creating task title="My task"

# 2. Chain client builds and signs transaction
[chain-client] CreateTask: building MsgCreateTask
[chain-client] GetAccount: sequence=5, account_number=0
[chain-client] BroadcastTx: sending signed tx

# 3. CometBFT receives and validates
INF received tx module=mempool tx=A1B2C3...
INF added good transaction module=mempool tx=A1B2C3...

# 4. Block is proposed and committed
INF finalizing commit of block height=100
INF committed state height=100 appHash=DEADBEEF...

# 5. API backend returns response
[api] POST /api/tasks: success tx_hash=A1B2C3...
```

### Filtering Logs

```bash
# Show only errors
docker compose logs 2>&1 | grep -i "err\|error\|panic\|fail"

# Show only API backend activity
docker compose logs api-backend -f

# Show only consensus activity
docker compose logs mychain-node -f 2>&1 | grep "committed\|finalizing"

# Show only specific request
docker compose logs api-backend -f 2>&1 | grep "POST /api/tasks"
```

---

## 6. Quick Recovery Cheat Sheet

| Problem                     | Quick Fix                                  |
|-----------------------------|--------------------------------------------|
| Everything broken           | `docker compose down -v && docker compose up --build -d` |
| One service crashed         | `docker compose restart <service>`         |
| Stale state                 | `docker compose down -v && docker compose up -d` |
| Port conflict               | Change port in docker-compose.yml          |
| Sequence mismatch           | Wait 5s and retry, or restart api-backend  |
| Contract not deployed       | Run deploy-contracts.sh                    |
| Can't reach endpoints       | Check `docker compose ps` for health       |
| Build fails                 | `docker compose build --no-cache`          |

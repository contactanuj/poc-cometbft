# ABCI KVStore Guide (Direct ABCI Application)

The ABCI KVStore is a **bare-bones key-value store** built directly against CometBFT's ABCI interface. It has no framework, no SDK, no abstractions -- just raw ABCI methods and a database.

> **Analogy**: If Cosmos SDK is building with a full construction framework (cranes, scaffolding, prefab walls), the ABCI KVStore is building with hand tools and raw materials. You understand every nail and beam because you placed them yourself.

---

## 1. Architecture

```
+----------------------------------------------------------+
|                   ABCI KVStore Container                  |
|                                                          |
|   User Request (PUT /api/kv/mykey)                       |
|        |                                                 |
|        v                                                 |
|   API Backend (:8080)                                    |
|        |                                                 |
|        | RPC: BroadcastTxCommit / ABCIQuery              |
|        v                                                 |
|   +----------------------------------------------+       |
|   |            CometBFT (:36657)                 |       |
|   |                                              |       |
|   |  Mempool -> Consensus -> Block Production    |       |
|   |                                              |       |
|   +-------------------+-----+--------------------+       |
|                       |     |                            |
|            ABCI socket|     |ABCI socket                 |
|             (:36658)  |     |(:36658)                    |
|                       v     v                            |
|   +----------------------------------------------+       |
|   |          KVStoreApp (Go binary)              |       |
|   |                                              |       |
|   |  +----------+    +----------+    +--------+  |       |
|   |  | CheckTx  |    |Finalize  |    | Query  |  |       |
|   |  | validate |    |Block     |    | read   |  |       |
|   |  | format   |    |stage     |    | from   |  |       |
|   |  +----------+    |changes   |    | Badger |  |       |
|   |                  +----+-----+    +--------+  |       |
|   |                       |                      |       |
|   |                  +----v-----+                |       |
|   |                  | Commit   |                |       |
|   |                  | persist  |                |       |
|   |                  | to Badger|                |       |
|   |                  +----+-----+                |       |
|   |                       |                      |       |
|   |                  +----v-----------+          |       |
|   |                  | Badger DB      |          |       |
|   |                  | /data/kvstore-db|         |       |
|   |                  +----------------+          |       |
|   +----------------------------------------------+       |
+----------------------------------------------------------+
```

**Key distinction from MyChain**: CometBFT and the KVStore app run as **separate processes** connected via an ABCI socket. In Cosmos SDK, CometBFT is embedded in the same process.

---

## 2. How ABCI Works

ABCI (Application Blockchain Interface) is a simple protocol with a small set of methods. CometBFT calls these methods on your application at specific points in the block lifecycle.

### Methods Implemented by KVStoreApp

| Method            | When Called                       | Purpose                                    |
|-------------------|-----------------------------------|--------------------------------------------|
| `Info`            | On startup / reconnect            | Return app version, height, appHash        |
| `CheckTx`        | When a tx enters the mempool      | Validate transaction format                |
| `PrepareProposal`| Proposer building a block          | Select/order transactions for the block    |
| `ProcessProposal`| Validators verifying a block       | Validate the proposed block                |
| `FinalizeBlock`  | After consensus agrees on a block  | Execute all transactions, stage changes    |
| `Commit`         | After FinalizeBlock                | Persist changes to disk, return appHash    |
| `Query`          | Anytime (client request)           | Read a value by key                        |

### Block Lifecycle Sequence

```
  CometBFT                              KVStoreApp
     |                                       |
     |  1. PrepareProposal(txs)              |
     |-------------------------------------->|
     |  Return: same txs (passthrough)       |
     |<--------------------------------------|
     |                                       |
     |  2. ProcessProposal(txs)              |
     |-------------------------------------->|
     |  Return: ACCEPT                       |
     |<--------------------------------------|
     |                                       |
     |  3. Consensus voting (prevote,        |
     |     precommit -- 2/3+ agreement)      |
     |                                       |
     |  4. FinalizeBlock(txs)                |
     |-------------------------------------->|
     |  For each tx:                         |
     |    Parse "key=value"                  |
     |    Stage in memory buffer             |
     |  Return: tx results                   |
     |<--------------------------------------|
     |                                       |
     |  5. Commit()                          |
     |-------------------------------------->|
     |  BatchSet(staged) -> Badger DB        |
     |  Hash() -> SHA256 of all KV pairs     |
     |  SaveMeta(height, appHash)            |
     |  Return: appHash                      |
     |<--------------------------------------|
     |                                       |
     |  Block finalized at height N          |
     |                                       |
```

---

## 3. Transaction Format

Transactions use a simple `key=value` text format:

```
mykey=hello world
```

### Validation Rules (CheckTx)

| Rule                     | Valid Example      | Invalid Example | Error                |
|--------------------------|--------------------|-----------------|----------------------|
| Must contain `=`         | `name=Alice`       | `nameAlice`     | "invalid tx format"  |
| Key must not be empty    | `k=v`              | `=value`        | "invalid tx format"  |
| Only one `=` matters     | `a=b=c`            | (this is valid) | key=`a`, value=`b=c` |

The first `=` splits key and value. Everything after the first `=` is the value (including additional `=` signs).

---

## 4. State Management

### Staged Writes Pattern

Changes are NOT written to the database immediately. They follow a two-phase pattern:

```
Phase 1: FinalizeBlock
  +----------------+
  | staged buffer  |  In-memory list of {key, value} pairs
  | ([]KVPair)     |  NOT yet in the database
  +----------------+

Phase 2: Commit
  +----------------+     +----------------+
  | staged buffer  | --> | Badger DB      |  BatchSet atomically writes all
  | (cleared)      |     | (persisted)    |  staged pairs to disk
  +----------------+     +----------------+
```

**Why this pattern?** If consensus fails between FinalizeBlock and Commit, the staged changes are discarded. Only committed state is real. This ensures the database always reflects a valid consensus state.

### App Hash

After every Commit, the app computes a SHA-256 hash of all key-value pairs in the database:

```
SHA256(
  key1 + value1 +
  key2 + value2 +
  ...
)
```

This hash is stored in the block header. Any node can verify that its state matches the network by comparing app hashes. If two nodes have different app hashes at the same height, one of them has a bug.

### Meta Storage

Two special keys are stored for crash recovery:

| Key               | Value                        | Purpose                      |
|-------------------|------------------------------|------------------------------|
| `__meta_height`   | 8-byte big-endian uint64     | Last committed block height  |
| `__meta_apphash`  | 32-byte SHA-256 hash         | Last committed app hash      |

On startup, `LoadMeta()` reads these to resume from the correct height. These keys are excluded from the app hash calculation.

---

## 5. API Reference

### PUT /api/kv/{key} -- Set a Key-Value Pair

Write a value for the given key. The write goes through full consensus.

**Request:**
```bash
curl -X PUT http://localhost:8080/api/kv/greeting \
  -H "Content-Type: application/json" \
  -d '{"value": "hello world"}'
```

| Parameter | Location | Type   | Required | Description     |
|-----------|----------|--------|----------|-----------------|
| `key`     | URL path | string | Yes      | The key to set  |
| `value`   | Body     | string | Yes      | The value       |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "key": "greeting",
    "value": "hello world"
  }
}
```

**Note:** Uses `BroadcastTxCommit` -- the response is returned only after the block is committed. This is slower (~5 seconds) but guarantees the write is persisted.

---

### GET /api/kv/{key} -- Get a Value

Read a value by key. This is a direct query -- no consensus needed.

**Request:**
```bash
curl http://localhost:8080/api/kv/greeting
```

| Parameter | Location | Type   | Required | Description     |
|-----------|----------|--------|----------|-----------------|
| `key`     | URL path | string | Yes      | The key to read |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "key": "greeting",
    "value": "hello world"
  }
}
```

**Response (key not found):**
```json
{
  "success": true,
  "data": {
    "key": "nonexistent",
    "value": ""
  }
}
```

---

## 6. Internal Flows

### Set Operation: Full Path

```
1. User:    PUT /api/kv/color  {"value": "blue"}

2. API:     Build tx bytes: "color=blue"

3. API:     RPC BroadcastTxCommit to :36657
                |
4. CometBFT:   CheckTx("color=blue")
                  -> contains "="? yes
                  -> key "color" non-empty? yes
                  -> code: 0 (accepted into mempool)
                |
5. CometBFT:   Block proposed with ["color=blue"]
                |
6. CometBFT:   FinalizeBlock(["color=blue"])
                  -> parse: key="color", value="blue"
                  -> staged = [{key: "color", value: "blue"}]
                  -> return code: 0
                |
7. CometBFT:   Commit()
                  -> state.BatchSet([{color, blue}])
                  -> state.Hash() -> SHA256 of all pairs
                  -> state.SaveMeta(height, appHash)
                  -> return appHash
                |
8. API:     Receives commit result, returns to user
```

### Get Operation: Direct Query

```
1. User:    GET /api/kv/color

2. API:     RPC ABCIQuery to :36657
              path: "", data: "color"
                |
3. CometBFT:   Routes to KVStoreApp.Query()
                |
4. KVStore:   state.Get("color")
                -> Badger DB read
                -> returns "blue"
                |
5. API:     Returns {"key": "color", "value": "blue"}
```

**Key difference**: Writes go through consensus (slow, ~5 seconds). Reads go directly to the database (fast, milliseconds).

---

## 7. CometBFT Configuration

The ABCI KVStore container runs both CometBFT and the KVStore app. The `entrypoint.sh` script configures them:

```bash
# Initialize CometBFT (creates config and genesis)
cometbft init

# Configure ABCI connection
sed -i 's|proxy_app = "tcp://127.0.0.1:26658"|proxy_app = "tcp://127.0.0.1:36658"|' config.toml

# Configure RPC (for external access)
sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:36657"|' config.toml

# Configure P2P
sed -i 's|laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:36656"|' config.toml

# Start KVStore app in background
/usr/local/bin/kvstore-app --addr tcp://0.0.0.0:36658 &
sleep 2

# Start CometBFT in foreground
cometbft start
```

**Port remapping**: Default CometBFT ports (26656/26657/26658) are changed to 36656/36657/36658 to avoid conflicting with MyChain.

---

## 8. Direct CometBFT RPC Access

You can bypass the API backend and talk directly to the KVStore's CometBFT instance:

```bash
# Check node status
curl http://localhost:36657/status | jq

# Broadcast a transaction directly
curl "http://localhost:36657/broadcast_tx_commit?tx=\"mykey=myvalue\""

# Query a key directly
curl "http://localhost:36657/abci_query?data=\"mykey\"" | jq

# The value is base64-encoded in the response:
# .result.response.value -> base64 decode -> "myvalue"
echo "bXl2YWx1ZQ==" | base64 -d
# -> myvalue
```

---

## 9. Step-by-Step Tutorial

```bash
# 1. Set some key-value pairs
curl -X PUT http://localhost:8080/api/kv/name \
  -H "Content-Type: application/json" \
  -d '{"value": "Alice"}'

curl -X PUT http://localhost:8080/api/kv/role \
  -H "Content-Type: application/json" \
  -d '{"value": "Engineer"}'

curl -X PUT http://localhost:8080/api/kv/team \
  -H "Content-Type: application/json" \
  -d '{"value": "Backend"}'

# 2. Read them back
curl http://localhost:8080/api/kv/name | jq    # -> "Alice"
curl http://localhost:8080/api/kv/role | jq    # -> "Engineer"
curl http://localhost:8080/api/kv/team | jq    # -> "Backend"

# 3. Update a value (same key, new value)
curl -X PUT http://localhost:8080/api/kv/role \
  -H "Content-Type: application/json" \
  -d '{"value": "Senior Engineer"}'

# 4. Read updated value
curl http://localhost:8080/api/kv/role | jq    # -> "Senior Engineer"

# 5. Read a non-existent key
curl http://localhost:8080/api/kv/nonexistent | jq   # -> value: ""

# 6. Verify directly via CometBFT RPC
curl "http://localhost:36657/abci_query?data=\"name\"" | jq '.result.response'
```

---

## 10. Source Code Reference

| File                            | Purpose                                     |
|---------------------------------|---------------------------------------------|
| `abci-kvstore/main.go`         | Entry point, flag parsing, server startup   |
| `abci-kvstore/app.go`          | ABCI interface implementation               |
| `abci-kvstore/state.go`        | Badger DB wrapper (Get, Set, Hash, Meta)    |
| `abci-kvstore/go.mod`          | Dependencies                                |
| `abci-kvstore/Dockerfile`      | Container build (app + CometBFT)            |
| `abci-kvstore/entrypoint.sh`   | Startup script (init + configure + start)   |
| `api-backend/client/kvstore.go` | KVStore RPC client                          |
| `api-backend/handlers/kvstore.go`| HTTP handlers for /api/kv/*                |

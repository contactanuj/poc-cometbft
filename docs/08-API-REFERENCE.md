# Complete API Reference

All endpoints served by the REST API Backend at `http://localhost:8080`.

---

## Conventions

### Base URL

```
http://localhost:8080
```

### Response Format

Every response is wrapped in a standard envelope:

```json
// Success
{
  "success": true,
  "data": { ... }
}

// Error
{
  "success": false,
  "error": "error message here"
}
```

### Content Type

- Request: `Content-Type: application/json`
- Response: `Content-Type: application/json`

### Middleware

| Middleware    | Behavior                                          |
|--------------|---------------------------------------------------|
| Logger       | Logs every request with method, path, duration    |
| Recoverer    | Catches panics, returns 500 instead of crashing   |
| Timeout      | Aborts requests after 30 seconds                  |

---

## Health

### GET /api/health

Check the health of all backend services.

```bash
curl http://localhost:8080/api/health
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "chain": "ok",
    "kvstore": "ok"
  }
}
```

Each service is checked independently by querying its CometBFT `/status` endpoint.

---

## Task Management (Cosmos SDK Module)

These endpoints interact with the `x/taskreg` custom Cosmos SDK module.

### POST /api/tasks

Create a new task.

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Fix login bug", "description": "Users cannot login"}'
```

**Request Body:**

| Field         | Type   | Required | Description       |
|---------------|--------|----------|-------------------|
| `title`       | string | Yes      | Task title        |
| `description` | string | No       | Task description  |

**Response (201):**
```json
{"success": true, "data": {"tx_hash": "A1B2C3D4..."}}
```

**Errors:** 400 (missing title), 500 (broadcast failed)

---

### GET /api/tasks

List all tasks.

```bash
curl http://localhost:8080/api/tasks
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "id": "1",
        "title": "Fix login bug",
        "description": "Users cannot login",
        "creator": "cosmos1...",
        "assignee": "",
        "status": "TASK_STATUS_OPEN"
      }
    ],
    "pagination": {"total": "1"}
  }
}
```

---

### GET /api/tasks/{id}

Get a specific task by ID.

```bash
curl http://localhost:8080/api/tasks/1
```

**URL Parameters:**

| Parameter | Type   | Description         |
|-----------|--------|---------------------|
| `id`      | uint64 | Task ID             |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "task": {
      "id": "1",
      "title": "Fix login bug",
      "description": "Users cannot login",
      "creator": "cosmos1...",
      "assignee": "",
      "status": "TASK_STATUS_OPEN"
    }
  }
}
```

**Errors:** 400 (invalid ID), 500 (not found)

---

### POST /api/tasks/{id}/assign

Assign a task to a worker. Only the task creator's signing key can do this (the API backend's key must be the creator).

```bash
curl -X POST http://localhost:8080/api/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d '{"assignee": "cosmos1cypznegvhp348zaqfcntmkvm0tg4pjc7kl4yxr"}'
```

**Request Body:**

| Field      | Type   | Required | Description            |
|------------|--------|----------|------------------------|
| `assignee` | string | Yes      | Cosmos address of worker|

**Response (200):**
```json
{"success": true, "data": {"tx_hash": "D4E5F6..."}}
```

**Errors:** 400 (invalid ID, missing assignee), 500 (unauthorized, wrong status)

---

### POST /api/tasks/{id}/complete

Mark a task as completed. Only the assignee's signing key can do this.

```bash
curl -X POST http://localhost:8080/api/tasks/1/complete
```

**Response (200):**
```json
{"success": true, "data": {"tx_hash": "G7H8I9..."}}
```

**Errors:** 400 (invalid ID), 500 (unauthorized, wrong status)

---

## Bounty Management (CosmWasm Contract)

These endpoints interact with the Task Bounty CosmWasm smart contract. Requires contract deployment first (see [06-BOUNTY-CONTRACT-GUIDE.md](06-BOUNTY-CONTRACT-GUIDE.md)).

### POST /api/bounties/fund

Fund a bounty for a task.

```bash
curl -X POST http://localhost:8080/api/bounties/fund \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1, "amount": "1000", "denom": "stake"}'
```

**Request Body:**

| Field     | Type   | Required | Description              |
|-----------|--------|----------|--------------------------|
| `task_id` | uint64 | Yes      | Task to fund bounty for  |
| `amount`  | string | Yes      | Token amount (as string) |
| `denom`   | string | Yes      | Token denomination       |

**Response (201):**
```json
{"success": true, "data": {"tx_hash": "J0K1L2..."}}
```

**Errors:** 500 (contract not deployed, duplicate bounty, no funds)

---

### POST /api/bounties/claim

Claim a bounty payout.

```bash
curl -X POST http://localhost:8080/api/bounties/claim \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1}'
```

**Request Body:**

| Field     | Type   | Required | Description           |
|-----------|--------|----------|-----------------------|
| `task_id` | uint64 | Yes      | Task to claim for     |

**Response (200):**
```json
{"success": true, "data": {"tx_hash": "M3N4O5..."}}
```

**Errors:** 500 (not found, already claimed)

---

### GET /api/bounties/{id}

Get bounty details by task ID.

```bash
curl http://localhost:8080/api/bounties/1
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "task_id": 1,
    "funder": "cosmos1...",
    "amount": "1000",
    "denom": "stake",
    "multiplier_pct": 100,
    "claimed": false,
    "claimer": null
  }
}
```

---

### GET /api/bounties

List all bounties (default: 10, max: 30 per page).

```bash
curl http://localhost:8080/api/bounties
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "task_id": 1,
      "funder": "cosmos1...",
      "amount": "1000",
      "denom": "stake",
      "multiplier_pct": 100,
      "claimed": false,
      "claimer": null
    }
  ]
}
```

---

### GET /api/config

Get the bounty contract configuration.

```bash
curl http://localhost:8080/api/config
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "admin": "cosmos1...",
    "fee_collector": "cosmos1...",
    "fee_pct": 10
  }
}
```

---

## Key-Value Store (Direct ABCI)

These endpoints interact with the ABCI KVStore application.

### PUT /api/kv/{key}

Set a key-value pair. Goes through full consensus (BroadcastTxCommit).

```bash
curl -X PUT http://localhost:8080/api/kv/mykey \
  -H "Content-Type: application/json" \
  -d '{"value": "hello"}'
```

**Parameters:**

| Parameter | Location | Type   | Required | Description |
|-----------|----------|--------|----------|-------------|
| `key`     | URL path | string | Yes      | The key     |
| `value`   | Body     | string | Yes      | The value   |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "key": "mykey",
    "value": "hello"
  }
}
```

**Note:** This uses `BroadcastTxCommit` which waits for the block to be committed (~5 seconds). The response confirms the write is durable.

---

### GET /api/kv/{key}

Get a value by key. Direct database read -- no consensus.

```bash
curl http://localhost:8080/api/kv/mykey
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "key": "mykey",
    "value": "hello"
  }
}
```

Returns empty value if key does not exist (does not return an error).

---

## Direct Chain Access (Bypass API Backend)

### CometBFT RPC (MyChain)

```bash
# Node status
curl http://localhost:26657/status | jq

# Latest block
curl http://localhost:26657/block | jq

# Search transactions by event
curl "http://localhost:26657/tx_search?query=\"tx.height>0\"" | jq

# Network info
curl http://localhost:26657/net_info | jq
```

### Cosmos SDK REST API (MyChain)

```bash
# List all accounts
curl http://localhost:1317/cosmos/auth/v1beta1/accounts | jq

# Get account balance
curl http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos1... | jq

# Query task via REST gateway
curl http://localhost:1317/mychain/taskreg/v1/task/1 | jq

# List tasks via REST gateway
curl http://localhost:1317/mychain/taskreg/v1/tasks | jq
```

### CometBFT RPC (KVStore)

```bash
# Node status
curl http://localhost:36657/status | jq

# Broadcast a transaction directly
curl "http://localhost:36657/broadcast_tx_commit?tx=\"key=value\""

# Query a key (value is base64-encoded in response)
curl "http://localhost:36657/abci_query?data=\"key\"" | jq
```

---

## Concurrency and Rate Limiting

### Transaction Serialization

The API backend uses a `sync.Mutex` to serialize all transaction broadcasts to MyChain. This prevents **sequence mismatch** errors.

**What is a sequence mismatch?**

Every Cosmos account has a sequence number (nonce) that increments with each transaction. If two transactions are signed with sequence 5, the second one will be rejected:

```
Tx A: signed with seq=5 -> accepted, seq becomes 6
Tx B: signed with seq=5 -> REJECTED (expected seq=6)
```

The mutex ensures transactions are signed and broadcast one at a time.

### Recommended Wait Times

| Operation          | Wait Time   | Why                                             |
|--------------------|-------------|--------------------------------------------------|
| After write (task) | ~5 seconds  | Wait for block commit before querying            |
| Between writes     | ~1 second   | Allow mutex to release and sequence to update    |
| After deploy       | ~10 seconds | Contract instantiation needs 2 blocks            |

### Error: "account sequence mismatch"

If you see this error, it means transactions were submitted too fast. Wait a few seconds and retry. The mutex prevents this in normal operation, but can happen under extreme load.

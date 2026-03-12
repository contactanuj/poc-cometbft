# Architecture Deep Dive

This document explains the system architecture, communication patterns, data flows, and design decisions.

> **Prerequisite**: Read [02-GLOSSARY.md](02-GLOSSARY.md) if any term is unfamiliar.

---

## 1. System Topology

```
+-----------------------------------------------------------------------+
|                          Docker Network: cometbft-net                  |
|                                                                       |
|  +---------------------+    +-----------------+    +-----------------+|
|  |  mychain-node        |    | abci-kvstore    |    | api-backend     ||
|  |  (Cosmos SDK + ABCI) |    | (ABCI + BFT)    |    | (Go HTTP)       ||
|  |                      |    |                  |    |                 ||
|  |  :26656  P2P         |    | :36656  P2P      |    | :8080  HTTP     ||
|  |  :26657  RPC         |    | :36657  RPC      |    |                 ||
|  |  :1317   REST        |    | :36658  ABCI     |    |                 ||
|  |  :9090   gRPC        |    |                  |    |                 ||
|  +---------------------+    +-----------------+    +-----------------+|
|         |                          |                    |    |    |    |
|         |  volumes:                |  volumes:          |    |    |    |
|         |  mychain-data            |  abci-data         |    |    |    |
|         |                          |                    |    |    |    |
+-----------------------------------------------------------------------+
          ^                          ^                    |    |    |
          |         gRPC :9090       |                    |    |    |
          +--------------------------------------------------+    |
          |         RPC  :26657      |                         |    |
          +--------------------------------------------------+    |
                                     |    RPC  :36657              |
                                     +-----------------------------+
```

### Container Dependencies (startup order)

```
mychain-node       starts first, health check: curl localhost:26657/status
       |
abci-kvstore       starts first, health check: curl localhost:36657/status
       |
api-backend        depends_on: mychain-node (healthy), abci-kvstore (healthy)
                    waits for both chains before starting
```

---

## 2. Component Internals

### MyChain Node -- What Runs Inside

```
+------------------------------------------------------------------+
|                        MyChain Application                        |
|                        (appName: "MyChainApp")                    |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |                    Ante Handler Chain                        |  |
|  |  SetUpContext -> ValidateBasic -> DeductFee -> VerifySig    |  |
|  |  -> IncrementSequence -> (+ WASM-specific decorators)       |  |
|  +------------------------------------------------------------+  |
|                                                                   |
|  +------------+ +----------+ +----------+ +----------+           |
|  | x/auth     | | x/bank   | | x/staking| | x/gov    |           |
|  | (accounts) | | (tokens) | | (valid.) | | (voting) |           |
|  +------------+ +----------+ +----------+ +----------+           |
|                                                                   |
|  +------------+ +----------+ +----------+ +----------+           |
|  | x/mint     | | x/distr  | | x/slashing| | x/evidence|         |
|  | (inflation)| | (rewards)| | (penalty)| | (fraud)  |           |
|  +------------+ +----------+ +----------+ +----------+           |
|                                                                   |
|  +------------+ +----------+ +----------------------------+      |
|  | x/ibc      | | x/transfer| | x/wasm (wasmd)            |      |
|  | (cross-ch) | | (IBC tkn)| | CosmWasm VM + contracts   |      |
|  +------------+ +----------+ +----------------------------+      |
|                                                                   |
|  +----------------------------+                                   |
|  | x/taskreg (CUSTOM)         |                                   |
|  | Task Registry module       |                                   |
|  +----------------------------+                                   |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |              CometBFT Consensus Engine (embedded)           |  |
|  |              P2P :26656  |  RPC :26657                      |  |
|  +------------------------------------------------------------+  |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |              IAVL Tree State Storage (LevelDB)              |  |
|  +------------------------------------------------------------+  |
+------------------------------------------------------------------+
```

**23 modules total**: auth, bank, staking, mint, distribution, governance, slashing, crisis, params, consensus, genutil, evidence, feegrant, upgrade, vesting, capability, IBC core, IBC transfer, IBC fee, wasm, and custom taskreg.

### ABCI KVStore -- What Runs Inside

```
+------------------------------------------+
|          ABCI KVStore Container           |
|                                          |
|  +------------------------------------+  |
|  |    KVStoreApp (Go)                 |  |
|  |                                    |  |
|  |  CheckTx()      - validate tx     |  |
|  |  FinalizeBlock() - stage changes   |  |
|  |  Commit()        - persist + hash  |  |
|  |  Query()         - read by key     |  |
|  |  Info()          - return metadata |  |
|  |                                    |  |
|  |  +-----------+  +---------------+  |  |
|  |  | staged[]  |  | Badger DB     |  |  |
|  |  | (in-mem)  |  | (persistent)  |  |  |
|  |  +-----------+  +---------------+  |  |
|  +------------------------------------+  |
|         ABCI socket :36658               |
|              |                           |
|  +------------------------------------+  |
|  |    CometBFT (separate process)     |  |
|  |    P2P :36656  |  RPC :36657       |  |
|  +------------------------------------+  |
+------------------------------------------+
```

### API Backend -- What Runs Inside

```
+--------------------------------------------------+
|              API Backend Container                |
|                                                  |
|  +--------------------------------------------+  |
|  |  Chi Router (HTTP :8080)                   |  |
|  |                                            |  |
|  |  Middleware: Logger, Recoverer, Timeout(30s)|  |
|  |                                            |  |
|  |  /api/health      -> HealthHandler         |  |
|  |  /api/tasks/*      -> TaskHandler          |  |
|  |  /api/bounties/*   -> BountyHandler        |  |
|  |  /api/kv/*         -> KVStoreHandler       |  |
|  |  /api/config       -> BountyHandler        |  |
|  +--------------------------------------------+  |
|                                                  |
|  +--------------------+  +--------------------+  |
|  | ChainClient        |  | KVStoreClient      |  |
|  |                    |  |                    |  |
|  | - gRPC conn :9090  |  | - RPC conn :36657  |  |
|  | - RPC client :26657|  |                    |  |
|  | - In-memory keyring|  | BroadcastTxCommit  |  |
|  | - HD wallet (BIP44)|  | ABCIQuery          |  |
|  | - Mutex for txs    |  +--------------------+  |
|  | - WASM query client|                          |
|  | - Auth query client|                          |
|  | - Task query client|                          |
|  +--------------------+                          |
+--------------------------------------------------+
```

---

## 3. Communication Patterns

### Protocol Choice: gRPC vs RPC

| Operation        | Protocol | Why                                                        |
|------------------|----------|------------------------------------------------------------|
| Query tasks      | gRPC     | Typed protobuf responses, fast, SDK-native query interface |
| Query bounties   | gRPC     | SmartContractState query via wasm query client             |
| Query accounts   | gRPC     | Auth module query for sequence numbers                     |
| Broadcast txs    | RPC      | CometBFT RPC for BroadcastTxSync (submit + wait for check)|
| Health check     | RPC      | CometBFT /status endpoint (simple HTTP)                   |
| KV store read    | RPC      | ABCIQuery via CometBFT RPC (no gRPC on KVStore)           |
| KV store write   | RPC      | BroadcastTxCommit via CometBFT RPC                        |

### Transaction Signing Flow

```
1. Handler receives HTTP request
          |
2. Build Cosmos SDK message (e.g., MsgCreateTask)
          |
3. Get signer address from in-memory keyring
          |
4. Fetch account info via gRPC (account number + sequence)
          |
5. Create TxFactory (gas: 500k, fees: "0stake", SIGN_MODE_DIRECT)
          |
6. Build unsigned transaction
          |
7. Sign with private key (Secp256k1)
          |
8. Encode to protobuf bytes
          |
9. Mutex lock (prevent concurrent sequence conflicts)
          |
10. Broadcast via RPC (BroadcastTxSync)
          |
11. Mutex unlock
          |
12. Return tx hash to caller
```

---

## 4. Request Flow Diagrams

### Flow 1: Creating a Task (Cosmos SDK Module)

```
User                API Backend           MyChain Node            CometBFT
 |                      |                      |                      |
 |  POST /api/tasks     |                      |                      |
 |  {title: "Fix bug"}  |                      |                      |
 |--------------------->|                      |                      |
 |                      |                      |                      |
 |                      |  gRPC: GetAccount()  |                      |
 |                      |--------------------->|                      |
 |                      |  {seq: 5, accNum: 0} |                      |
 |                      |<---------------------|                      |
 |                      |                      |                      |
 |                      |  Sign MsgCreateTask  |                      |
 |                      |  (local, in-memory)  |                      |
 |                      |                      |                      |
 |                      |  RPC: BroadcastTxSync               |      |
 |                      |------------------------------------------>|
 |                      |                      |                      |
 |                      |                      |  CheckTx (validate)  |
 |                      |                      |<---------------------|
 |                      |                      |  -> mempool           |
 |                      |                      |                      |
 |                      |  {code: 0, hash: "A1B2..."}                |
 |                      |<------------------------------------------|
 |                      |                      |                      |
 |  {success: true,     |                      |                      |
 |   data: {tx_hash}}   |                      |  PrepareProposal     |
 |<---------------------|                      |<---------------------|
 |                      |                      |  ProcessProposal     |
 |                      |                      |<---------------------|
 |                      |                      |  FinalizeBlock       |
 |                      |                      |<---------------------|
 |                      |                      |  -> msg_server.      |
 |                      |                      |     CreateTask()     |
 |                      |                      |  -> keeper.SetTask() |
 |                      |                      |                      |
 |                      |                      |  Commit              |
 |                      |                      |<---------------------|
 |                      |                      |  -> persist to IAVL  |
```

### Flow 2: Funding a Bounty (CosmWasm Contract)

```
User                API Backend           MyChain Node            CosmWasm VM
 |                      |                      |                      |
 |  POST /api/bounties  |                      |                      |
 |  /fund               |                      |                      |
 |  {task_id: 1,        |                      |                      |
 |   amount: "1000",    |                      |                      |
 |   denom: "stake"}    |                      |                      |
 |--------------------->|                      |                      |
 |                      |                      |                      |
 |                      |  Build MsgExecuteContract                   |
 |                      |  contract: cosmos14hj...                    |
 |                      |  msg: {"fund_bounty":{"task_id":1}}        |
 |                      |  funds: [1000stake]                        |
 |                      |                      |                      |
 |                      |  Sign + Broadcast    |                      |
 |                      |--------------------->|                      |
 |                      |                      |                      |
 |                      |                      |  FinalizeBlock       |
 |                      |                      |  -> x/wasm keeper    |
 |                      |                      |  -> execute contract |
 |                      |                      |--------------------->|
 |                      |                      |                      |
 |                      |                      |  execute_fund_bounty:|
 |                      |                      |  - validate 1 denom  |
 |                      |                      |  - check no dup      |
 |                      |                      |  - save Bounty{      |
 |                      |                      |      funder, amount, |
 |                      |                      |      multiplier:100} |
 |                      |                      |<---------------------|
 |                      |                      |                      |
 |                      |                      |  Commit              |
 |                      |                      |                      |
 |  {success: true,     |                      |                      |
 |   data: {tx_hash}}   |                      |                      |
 |<---------------------|                      |                      |
```

### Flow 3: Setting a Key-Value Pair (Direct ABCI)

```
User                API Backend           CometBFT (:36657)      KVStoreApp
 |                      |                      |                      |
 |  PUT /api/kv/mykey   |                      |                      |
 |  {value: "hello"}    |                      |                      |
 |--------------------->|                      |                      |
 |                      |                      |                      |
 |                      |  RPC: BroadcastTxCommit                     |
 |                      |  tx: "mykey=hello"   |                      |
 |                      |--------------------->|                      |
 |                      |                      |                      |
 |                      |                      |  CheckTx             |
 |                      |                      |--------------------->|
 |                      |                      |  validate: has "="   |
 |                      |                      |  key non-empty: yes  |
 |                      |                      |<-- code: 0 (ok)      |
 |                      |                      |                      |
 |                      |                      |  FinalizeBlock       |
 |                      |                      |--------------------->|
 |                      |                      |  stage: mykey=hello  |
 |                      |                      |<-- code: 0           |
 |                      |                      |                      |
 |                      |                      |  Commit              |
 |                      |                      |--------------------->|
 |                      |                      |  BatchSet(staged)    |
 |                      |                      |  Hash() -> appHash   |
 |                      |                      |  SaveMeta(h, hash)   |
 |                      |                      |<-- appHash           |
 |                      |                      |                      |
 |                      |  {code: 0}           |                      |
 |                      |<---------------------|                      |
 |                      |                      |                      |
 |  {success: true,     |                      |                      |
 |   data: {key, value}}|                      |                      |
 |<---------------------|                      |                      |
```

### Flow 4: Query Path (No Consensus)

```
User                API Backend           MyChain Node
 |                      |                      |
 |  GET /api/tasks/5    |                      |
 |--------------------->|                      |
 |                      |                      |
 |                      |  gRPC: QueryTask(5)  |
 |                      |--------------------->|
 |                      |                      |
 |                      |  (reads directly     |
 |                      |   from IAVL store,   |
 |                      |   NO consensus, NO   |
 |                      |   block production)  |
 |                      |                      |
 |                      |  {task: {id:5, ...}} |
 |                      |<---------------------|
 |                      |                      |
 |  {success: true,     |                      |
 |   data: {task}}      |                      |
 |<---------------------|                      |
```

---

## 5. ABCI Block Lifecycle

This is the sequence CometBFT follows for **every block**:

```
                    +-------------------+
                    |   New Round       |
                    | (select proposer) |
                    +---------+---------+
                              |
                    +---------v---------+
                    | PrepareProposal   |
                    | Proposer selects  |
                    | txs from mempool  |
                    | and orders them   |
                    +---------+---------+
                              |
                    +---------v---------+
                    | ProcessProposal   |
                    | All validators    |
                    | verify the block  |
                    | is valid          |
                    +---------+---------+
                              |
                    +---------v---------+
                    |   Prevote         |
                    |   (2/3+ vote)     |
                    +---------+---------+
                              |
                    +---------v---------+
                    |   Precommit       |
                    |   (2/3+ vote)     |
                    +---------+---------+
                              |
                    +---------v---------+
                    | FinalizeBlock     |
                    | Execute all txs   |
                    | Compute state     |
                    | changes           |
                    +---------+---------+
                              |
                    +---------v---------+
                    | Commit            |
                    | Persist to disk   |
                    | Return appHash    |
                    +---------+---------+
                              |
                    +---------v---------+
                    |   Block Finalized |
                    |   (irreversible)  |
                    +-------------------+
```

---

## 6. Cosmos SDK Module Architecture

How a message flows through the Cosmos SDK from receipt to state change:

```
  Incoming Transaction
         |
  +------v------+
  | Ante Handler |  Validate signature, check gas, deduct fees,
  | Chain        |  increment sequence, WASM-specific decorators
  +------+------+
         |
  +------v------+
  | Router       |  Match message type to the correct module
  |              |  MsgCreateTask -> x/taskreg
  |              |  MsgExecuteContract -> x/wasm
  +------+------+
         |
  +------v------+
  | Msg Server   |  Module's message handler (msg_server.go)
  |              |  Validates business logic
  |              |  Calls keeper methods
  +------+------+
         |
  +------v------+
  | Keeper       |  State access layer (keeper.go)
  |              |  SetTask(), GetTask(), etc.
  |              |  Reads/writes KV store
  +------+------+
         |
  +------v------+
  | KV Store     |  IAVL Merkle tree
  | (IAVL)       |  Prefix-based key namespacing
  |              |  e.g., "Task/value/{id}" -> Task bytes
  +-------------+
```

---

## 7. State Management Comparison

| Aspect              | MyChain (Cosmos SDK)        | KVStore (ABCI)              | Bounty Contract (CosmWasm) |
|---------------------|-----------------------------|-----------------------------|----------------------------|
| **Storage engine**  | IAVL tree (LevelDB)        | Badger DB                   | cw-storage-plus (on IAVL)  |
| **Data format**     | Protobuf-encoded bytes      | Raw bytes                   | JSON-serialized structs     |
| **Key structure**   | `ModulePrefix/Key/ID`       | Any string                  | Namespaced by Item/Map      |
| **State proof**     | Merkle proof from IAVL      | SHA256 of all pairs         | Inherits from IAVL          |
| **Namespacing**     | Per-module store keys       | None (flat namespace)       | Per-contract prefix         |
| **Querying**        | gRPC typed queries          | ABCIQuery (raw bytes)       | SmartContractState (JSON)   |
| **Iteration**       | Prefix iteration            | Badger iterator             | cw-storage-plus range()     |

---

## 8. Three Approaches Compared

| Criteria            | Cosmos SDK Module     | CosmWasm Contract       | Direct ABCI            |
|---------------------|-----------------------|-------------------------|------------------------|
| **Language**        | Go                    | Rust (-> WASM)          | Go                     |
| **Complexity**      | High                  | Medium                  | Low                    |
| **Performance**     | Highest (native)      | Good (near-native WASM) | Highest (no framework) |
| **Upgrade path**    | Chain halt + upgrade  | Contract migration      | Chain halt + upgrade   |
| **Isolation**       | None (runs in-process)| Sandboxed (WASM VM)     | Separate process       |
| **Tooling**         | Rich (protobuf, CLI)  | Rich (cargo, schemas)   | DIY                    |
| **State access**    | Direct KV store       | Via CosmWasm API only   | Direct DB access       |
| **Chain restart**   | Required for changes  | Not required             | Required for changes   |
| **Best for**        | Core chain features   | Business logic / DApps  | Learning / experiments |
| **Real-world use**  | Bank, staking, auth   | DEX, NFT, lending       | Custom consensus apps  |

---

## 9. Design Decisions and Trade-offs

### Why a Separate API Backend?

**Decision**: A standalone Go HTTP server instead of using the built-in Cosmos REST API.

**Reasoning**:
- Unifies three different blockchain interfaces (gRPC, CometBFT RPC, ABCI) under one REST API
- Provides a familiar HTTP/JSON interface for frontend developers
- Handles transaction signing server-side (the user does not need a wallet)
- Demonstrates real-world API gateway patterns for blockchain backends

**Trade-off**: Extra hop for every request. In production, clients might talk to the chain directly for performance.

### Why Single Validator?

**Decision**: One validator node with test mnemonics.

**Reasoning**: This is a POC. Multiple validators add operational complexity without teaching new concepts. The consensus mechanics are identical.

**Trade-off**: No fault tolerance. See [09-PRODUCTION-GUIDE.md](09-PRODUCTION-GUIDE.md) for multi-validator setup.

### Why Test Keyring with Fixed Mnemonics?

**Decision**: Hardcoded mnemonics in init.sh and API backend.

**Reasoning**: Reproducible development environment. Every `docker compose up` creates the same accounts with the same addresses.

**Trade-off**: Zero security. Anyone who reads the source code can sign transactions. **Never use in production.**

### Why Mutex on Transaction Broadcasting?

**Decision**: A `sync.Mutex` protects the `broadcastTx` method.

**Reasoning**: Cosmos accounts use a sequence number (nonce) that must increment by exactly 1 per transaction. If two transactions are signed with the same sequence, one will be rejected. The mutex ensures sequential signing.

**Trade-off**: Limits throughput to one transaction at a time per signing account. See [09-PRODUCTION-GUIDE.md](09-PRODUCTION-GUIDE.md) for scaling strategies.

### Why BroadcastTxSync Instead of BroadcastTxCommit?

**Decision**: The API backend uses `BroadcastTxSync` for chain transactions (returns after mempool acceptance) but `BroadcastTxCommit` for KVStore (waits for block commit).

**Reasoning**: Sync is faster -- the user gets a tx hash immediately. The transaction will be included in a future block. Commit waits for the full block cycle, which is slower but guarantees the write is persisted.

**Trade-off**: With Sync, the task is not yet on-chain when the API returns. The user must wait ~5 seconds (one block) before querying the task.

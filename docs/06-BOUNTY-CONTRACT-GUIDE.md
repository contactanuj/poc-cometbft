# Task Bounty Contract Guide (CosmWasm)

The Task Bounty contract is a **CosmWasm smart contract** written in Rust that manages financial incentives (bounties) for tasks. It demonstrates how to build upgradable business logic on a Cosmos SDK chain without modifying the chain itself.

> **Analogy**: A vending machine for bounties. Put money in (fund), someone does the work and presses the claim button, and the machine pays out -- automatically deducting a platform fee. No human middleman needed.

---

## 1. How CosmWasm Contracts Work

```
+------------------------------------------------------------------+
|  Developer writes Rust code                                       |
|  cargo build --target wasm32-unknown-unknown                      |
+----------------------------------+-------------------------------+
                                   |
                                   v
+------------------------------------------------------------------+
|  task_bounty.wasm (WebAssembly bytecode)                         |
+----------------------------------+-------------------------------+
                                   |
                       mychaind tx wasm store
                                   |
                                   v
+------------------------------------------------------------------+
|  Stored on-chain with Code ID = 1                                |
+----------------------------------+-------------------------------+
                                   |
                       mychaind tx wasm instantiate
                       {fee_collector: "cosmos1...", fee_pct: 10}
                                   |
                                   v
+------------------------------------------------------------------+
|  Contract Instance                                                |
|  Address: cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4...    |
|  Admin: deployer address                                          |
|  State: {config: {...}, bounties: {}}                            |
+------------------------------------------------------------------+
```

**Key concept**: Code ID is like a class. Contract address is like an object instance. You can create multiple instances from the same code.

---

## 2. Bounty Lifecycle

```
                   FundBounty
                   (anyone, with tokens)
                        |
                        v
                  +-----------+
                  |  FUNDED   |  Tokens locked in contract
                  |           |  multiplier_pct = 100 (1x)
                  +-----+-----+
                        |
                   AddBonus (optional)
                   (admin only)
                        |
                        v
                  +-----------+
                  |  BOOSTED  |  multiplier_pct > 100
                  |  (still   |  (e.g., 150 = 1.5x payout)
                  |  funded)  |
                  +-----+-----+
                        |
                   ClaimBounty
                   (anyone)
                        |
                        v
                  +-----------+
                  |  CLAIMED  |  Tokens paid out:
                  |           |    net -> claimer
                  |           |    fee -> fee_collector
                  +-----------+
```

---

## 3. Payout Calculation

### The Formula

```
gross = amount * multiplier_pct / 100
fee   = gross * fee_pct / 100
net   = gross - fee
```

### Worked Example

A bounty is funded with **1000 stake** tokens. The admin adds a **1.5x bonus** (multiplier_pct = 150). The platform fee is **10%** (fee_pct = 10).

```
Step 1: gross = 1000 * 150 / 100 = 1500 tokens
Step 2: fee   = 1500 * 10  / 100 = 150 tokens
Step 3: net   = 1500 - 150        = 1350 tokens

Result:
  Claimer receives:       1350 stake
  Fee collector receives:  150 stake
  Total disbursed:        1500 stake
```

```
+-------------------+
| Funded: 1000      |
| Multiplier: 150%  |
+--------+----------+
         |
         v
+-------------------+
| Gross: 1500       |
+--------+----------+
         |
    +----+--------+
    |             |
    v             v
+---------+  +---------+
| Net:    |  | Fee:    |
| 1350    |  | 150     |
| -> user |  | -> fee  |
|         |  | collector|
+---------+  +---------+
```

> **Important**: If the multiplier increases the payout beyond the originally funded amount, the extra tokens must already be in the contract's balance. The contract sends tokens from its own balance.

---

## 4. Data Models

### Config

```rust
pub struct Config {
    pub admin: Addr,           // Contract administrator (deployer)
    pub fee_collector: Addr,   // Address receiving platform fees
    pub fee_pct: u64,          // Fee percentage (0-100)
}
```

### Bounty

```rust
pub struct Bounty {
    pub task_id: u64,          // Which task this bounty is for
    pub funder: Addr,          // Who funded it
    pub amount: Uint128,       // Base amount (before multiplier)
    pub denom: String,         // Token denomination (e.g., "stake")
    pub multiplier_pct: u64,   // Bonus multiplier (default: 100 = 1x)
    pub claimed: bool,         // Has it been claimed?
    pub claimer: Option<Addr>, // Who claimed it (None if unclaimed)
}
```

### Storage

```
CONFIG        -> Item<Config>           (single value, JSON)
BOUNTIES      -> Map<u64, Bounty>       (task_id -> Bounty, JSON)
```

---

## 5. API Reference

### POST /api/bounties/fund -- Fund a Bounty

Lock tokens for a task bounty.

**Request:**
```bash
curl -X POST http://localhost:8080/api/bounties/fund \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 1,
    "amount": "1000",
    "denom": "stake"
  }'
```

| Field     | Type   | Required | Description                    |
|-----------|--------|----------|--------------------------------|
| `task_id` | uint64 | Yes      | Task to fund bounty for        |
| `amount`  | string | Yes      | Amount of tokens (as string)   |
| `denom`   | string | Yes      | Token denomination              |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "tx_hash": "A1B2C3..."
  }
}
```

**Errors:**
| Condition                          | Contract Error          |
|------------------------------------|-------------------------|
| No tokens sent with transaction    | `NoFunds`               |
| Multiple token denominations sent  | `MultipleDenoms`        |
| Bounty already exists for task_id  | `BountyAlreadyExists`   |

---

### POST /api/bounties/claim -- Claim a Bounty

Claim the bounty payout for a task.

**Request:**
```bash
curl -X POST http://localhost:8080/api/bounties/claim \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 1
  }'
```

| Field     | Type   | Required | Description         |
|-----------|--------|----------|---------------------|
| `task_id` | uint64 | Yes      | Task to claim for   |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "tx_hash": "D4E5F6..."
  }
}
```

**Errors:**
| Condition               | Contract Error        |
|-------------------------|-----------------------|
| Bounty does not exist   | `BountyNotFound`      |
| Already claimed         | `BountyAlreadyClaimed`|

---

### GET /api/bounties/{id} -- Get Bounty Details

**Request:**
```bash
curl http://localhost:8080/api/bounties/1
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "task_id": 1,
    "funder": "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za",
    "amount": "1000",
    "denom": "stake",
    "multiplier_pct": 100,
    "claimed": false,
    "claimer": null
  }
}
```

---

### GET /api/bounties -- List All Bounties

**Request:**
```bash
curl http://localhost:8080/api/bounties
```

**Response (200 OK):**
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

Pagination: Default 10 items, max 30 per page.

---

### GET /api/config -- Get Contract Configuration

**Request:**
```bash
curl http://localhost:8080/api/config
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "admin": "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za",
    "fee_collector": "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za",
    "fee_pct": 10
  }
}
```

---

## 6. Contract Deployment

### Using the Automated Script (Docker)

```bash
docker compose exec mychain-node bash /scripts/deploy-contracts.sh
```

What the script does:

```bash
# 1. Wait for chain readiness
until curl -s http://localhost:26657/status > /dev/null; do sleep 1; done

# 2. Store the WASM bytecode
mychaind tx wasm store /contracts/artifacts/task_bounty.wasm \
  --from admin --gas auto --gas-adjustment 1.3 -y

# 3. Get the Code ID (should be 1)
CODE_ID=$(mychaind query wasm list-code --output json | jq -r '.code_infos[-1].code_id')

# 4. Instantiate the contract
INIT_MSG='{"fee_collector":"cosmos1...","fee_pct":10}'
mychaind tx wasm instantiate $CODE_ID "$INIT_MSG" \
  --label "task-bounty" --no-admin --from admin -y

# 5. Get the contract address
CONTRACT=$(mychaind query wasm list-contract-by-code $CODE_ID --output json \
  | jq -r '.contracts[-1]')
```

### Manual Deployment (Local)

```bash
# 1. Build the contract
cd contracts/task-bounty
cargo build --release --target wasm32-unknown-unknown

# 2. (Optional) Optimize the WASM binary
docker run --rm -v "$(pwd)":/code cosmwasm/optimizer:0.16.0

# 3. Store on chain
mychaind tx wasm store artifacts/task_bounty.wasm \
  --from admin --keyring-backend test --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.3 -y

# 4. Query the Code ID
mychaind query wasm list-code

# 5. Instantiate
mychaind tx wasm instantiate 1 \
  '{"fee_collector":"cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za","fee_pct":10}' \
  --label "task-bounty" --no-admin \
  --from admin --keyring-backend test --chain-id mychain-1 -y

# 6. Get the contract address
mychaind query wasm list-contract-by-code 1
```

---

## 7. Error Reference

| Error                | When It Happens                                       |
|----------------------|-------------------------------------------------------|
| `Unauthorized`       | Non-admin calls UpdateFee or AddBonus                 |
| `BountyAlreadyExists`| FundBounty for a task_id that already has a bounty    |
| `BountyNotFound`     | ClaimBounty, AddBonus, or GetBounty for nonexistent   |
| `BountyAlreadyClaimed`| ClaimBounty on a bounty that was already claimed      |
| `NoFunds`            | FundBounty called without attaching tokens            |
| `MultipleDenoms`     | FundBounty called with more than one coin type        |
| `InvalidFee`         | InstantiateMsg or UpdateFee with fee_pct > 100        |

---

## 8. Internal Flow: How Fund and Claim Work

### FundBounty Execution Inside the Contract

```
MsgExecuteContract{
  Sender: "cosmos1abc...",
  Contract: "cosmos14hj...",
  Msg: {"fund_bounty": {"task_id": 1}},
  Funds: [Coin{denom: "stake", amount: "1000"}]
}
    |
    v
x/wasm module -> wasmvm -> contract execute()
    |
    v
execute_fund_bounty():
    |
    +-- info.funds.len() == 1?  NO -> Err(MultipleDenoms) or Err(NoFunds)
    |                           YES -> continue
    |
    +-- BOUNTIES.has(task_id)?  YES -> Err(BountyAlreadyExists)
    |                           NO  -> continue
    |
    +-- Create Bounty{
    |     task_id: 1,
    |     funder: "cosmos1abc...",
    |     amount: 1000,
    |     denom: "stake",
    |     multiplier_pct: 100,
    |     claimed: false,
    |     claimer: None,
    |   }
    |
    +-- BOUNTIES.save(task_id, bounty)
    |
    +-- Return Ok(Response::new()
          .add_attribute("action", "fund_bounty")
          .add_attribute("task_id", "1"))
```

### ClaimBounty Execution Inside the Contract

```
MsgExecuteContract{
  Sender: "cosmos1xyz...",
  Contract: "cosmos14hj...",
  Msg: {"claim_bounty": {"task_id": 1}},
  Funds: []
}
    |
    v
execute_claim_bounty():
    |
    +-- BOUNTIES.load(task_id)?  NOT FOUND -> Err(BountyNotFound)
    |                            FOUND     -> continue
    |
    +-- bounty.claimed?          YES -> Err(BountyAlreadyClaimed)
    |                            NO  -> continue
    |
    +-- Calculate payout:
    |     gross = 1000 * 100 / 100 = 1000
    |     fee   = 1000 * 10  / 100 = 100
    |     net   = 1000 - 100        = 900
    |
    +-- bounty.claimed = true
    +-- bounty.claimer = Some("cosmos1xyz...")
    +-- BOUNTIES.save(task_id, bounty)
    |
    +-- Return Ok(Response::new()
          .add_message(BankMsg::Send{          // Send 900 to claimer
              to_address: "cosmos1xyz...",
              amount: [Coin{denom: "stake", amount: "900"}]
          })
          .add_message(BankMsg::Send{          // Send 100 to fee collector
              to_address: fee_collector,
              amount: [Coin{denom: "stake", amount: "100"}]
          }))
```

---

## 9. Step-by-Step Tutorial

### Prerequisites

- Stack is running (`docker compose up --build -d`)
- Contract is deployed (`docker compose exec mychain-node bash /scripts/deploy-contracts.sh`)

### Full Bounty Lifecycle

```bash
# 1. Check contract config
curl http://localhost:8080/api/config | jq

# 2. Create a task first (bounties reference tasks by ID)
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Build feature X"}'
sleep 5

# 3. Fund a bounty for task 1 with 1000 stake
curl -X POST http://localhost:8080/api/bounties/fund \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1, "amount": "1000", "denom": "stake"}'
sleep 5

# 4. Check the bounty
curl http://localhost:8080/api/bounties/1 | jq
# Should show: amount=1000, multiplier_pct=100, claimed=false

# 5. List all bounties
curl http://localhost:8080/api/bounties | jq

# 6. Claim the bounty
curl -X POST http://localhost:8080/api/bounties/claim \
  -H "Content-Type: application/json" \
  -d '{"task_id": 1}'
sleep 5

# 7. Check the bounty again
curl http://localhost:8080/api/bounties/1 | jq
# Should show: claimed=true, claimer="cosmos1..."
```

---

## 10. Source Code Reference

| File                                          | Purpose                          |
|-----------------------------------------------|----------------------------------|
| `contracts/task-bounty/src/contract.rs`       | All handlers (instantiate, execute, query) |
| `contracts/task-bounty/src/msg.rs`            | Message definitions              |
| `contracts/task-bounty/src/state.rs`          | State structures and storage     |
| `contracts/task-bounty/src/error.rs`          | Custom error types               |
| `contracts/task-bounty/src/lib.rs`            | Module exports                   |
| `contracts/task-bounty/Cargo.toml`            | Dependencies                     |
| `api-backend/handlers/bounties.go`            | HTTP handlers for bounty endpoints |
| `api-backend/client/chain.go`                 | Contract interaction (fund, claim, query) |
| `deploy-contracts.sh`                         | Deployment script                |

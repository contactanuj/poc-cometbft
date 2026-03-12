# Glossary of Terms

Every term explained with a **technical definition**, an **everyday analogy**, and a **practical example from this project**.

---

## 1. Blockchain Fundamentals

### Blockchain

- **Technical**: A distributed, append-only data structure where blocks of transactions are cryptographically linked in sequence. Each block contains a hash of the previous block, making tampering detectable.
- **Everyday**: A shared Google Sheet where every row references the row before it. Everyone has a copy, and you can only add rows -- never edit or delete old ones.
- **In this project**: MyChain is a blockchain. Every task creation, bounty funding, and KV store write becomes a transaction in a block.

### Block

- **Technical**: A batch of transactions grouped together, validated by consensus, and appended to the chain. Contains a header (height, timestamp, previous hash) and a body (list of transactions).
- **Everyday**: A page in a ledger book. Each page has a page number, a date, and a list of entries.
- **In this project**: MyChain produces a block roughly every 5 seconds. Each block can contain multiple task/bounty/KV transactions.

### Transaction (Tx)

- **Technical**: A signed message that requests a state change. It includes the sender's signature, the operation to perform, and fees. It must be validated before execution.
- **Everyday**: A signed check that says "transfer $100 from account A to account B". The bank verifies the signature before processing.
- **In this project**: `MsgCreateTask{title: "Fix bug"}` is a transaction. So is `FundBounty{task_id: 1}`. The API backend signs and broadcasts these for you.

### Consensus

- **Technical**: The protocol by which all nodes in the network agree on the order and validity of transactions. CometBFT uses BFT (Byzantine Fault Tolerant) consensus -- blocks are finalized when 2/3+ of validators agree.
- **Everyday**: A group vote where a decision passes only when more than two-thirds of voters agree. Even if some voters lie or are absent, the group still reaches a correct decision.
- **In this project**: MyChain uses CometBFT consensus. In this POC there is one validator, so consensus is instant. In production, you would have 4+ validators.

### Node

- **Technical**: A computer running the blockchain software. It maintains a copy of the entire state and participates in the network. Nodes relay transactions and blocks to each other.
- **Everyday**: One of many identical filing cabinets in different offices, all kept in sync.
- **In this project**: The `mychain-node` Docker container is a node. The `abci-kvstore` container is also a node (for a separate, simpler chain).

### Validator

- **Technical**: A node that actively participates in consensus by proposing and voting on blocks. Validators stake tokens as collateral -- misbehavior results in slashing (losing tokens).
- **Everyday**: An election official who counts votes. They put up a deposit, and if they cheat, they lose it.
- **In this project**: The `admin` account created during chain init is the sole validator. It stakes tokens and proposes every block.

### Hash

- **Technical**: A fixed-size output produced by a cryptographic hash function (e.g., SHA-256). Any change to the input produces a completely different hash. Used for data integrity verification.
- **Everyday**: A fingerprint. Every person has a unique one, and you can use it to verify identity without seeing the whole person.
- **In this project**: The ABCI KVStore computes a SHA-256 hash of all key-value pairs after each block. This `appHash` is stored in the block header to prove state integrity.

### State

- **Technical**: The current snapshot of all data stored by the blockchain application. For a bank, state is all account balances. For a task manager, state is all tasks and their statuses.
- **Everyday**: The current balance sheet of a company -- it reflects everything that has happened up to now.
- **In this project**: MyChain's state includes all tasks (x/taskreg), all bounties (CosmWasm contract storage), account balances, and validator info. The KVStore's state is its Badger database contents.

### Genesis Block / Genesis File

- **Technical**: The first block in the chain (height 0). The genesis file (`genesis.json`) defines the initial state: accounts, balances, validator set, module parameters, and chain ID.
- **Everyday**: The constitution of a new country -- it defines the initial rules, who has what, and how things start.
- **In this project**: `mychain/scripts/init.sh` creates the genesis file with two accounts (`admin` and `alice`), each receiving 1 billion `stake` and 1 billion `token`.

### Gas

- **Technical**: A unit measuring computational effort required to execute a transaction. Each operation (storage read, write, signature verification) costs a specific amount of gas. Users pay fees proportional to gas consumed.
- **Everyday**: Postage stamps for a letter. A heavier letter (more complex transaction) needs more stamps.
- **In this project**: Transactions use 500,000 gas. Minimum gas price is set to "0stake" (free in this POC). In production, you would set real gas prices to prevent spam.

### Fees

- **Technical**: Payment attached to a transaction to compensate validators for processing it. Calculated as `gas_used * gas_price`. Collected by the fee collector module and distributed to validators.
- **Everyday**: The service charge on a money transfer. The bank processes your transfer and keeps a small fee.
- **In this project**: The POC uses zero fees (`"0stake"`). The bounty contract has its own fee mechanism (fee_pct) that is separate from chain gas fees.

### Mempool

- **Technical**: A waiting area where validated-but-not-yet-included transactions sit until a validator picks them up for the next block. Each node maintains its own mempool.
- **Everyday**: The queue at a post office. Letters wait in a basket until the mail carrier picks them up for the next delivery run.
- **In this project**: When the API backend broadcasts a transaction, it first enters the mempool (CheckTx validates it), then gets included in the next block.

### Finality

- **Technical**: The guarantee that a committed block will never be reverted. CometBFT provides **instant finality** -- once a block is committed (2/3+ votes), it is permanent.
- **Everyday**: When a court ruling becomes final and cannot be appealed. In CometBFT, every block is immediately final.
- **In this project**: Once a task creation transaction is committed, it is permanent. Unlike Bitcoin or Ethereum (which have probabilistic finality), CometBFT blocks are irreversible immediately.

### Determinism

- **Technical**: The property that given the same input and state, every node produces exactly the same output. This is critical -- if nodes disagree on state, the chain forks.
- **Everyday**: A recipe that produces the exact same cake every time, no matter who follows it, as long as they use the same ingredients.
- **In this project**: Both the Cosmos SDK modules and CosmWasm contracts must be deterministic. This is why WASM is used for contracts -- it guarantees identical execution on every node.

### Fork

- **Technical**: A divergence in the chain where two or more blocks are produced at the same height. Can be a protocol upgrade (hard fork) or a consensus failure. CometBFT's instant finality makes accidental forks extremely unlikely.
- **Everyday**: A road splitting into two paths. Everyone must follow the same path, or the group splits.
- **In this project**: Not applicable in normal operation with CometBFT's BFT consensus. Protocol upgrades would be coordinated via the governance module.

---

## 2. CometBFT / Tendermint

### CometBFT

- **Technical**: A Byzantine Fault Tolerant consensus engine that handles networking and consensus. Applications communicate with it via ABCI. Formerly called Tendermint Core; rebranded to CometBFT in 2023.
- **Everyday**: The engine of a car. You build the car body (your app), and CometBFT provides the engine that makes it go.
- **In this project**: CometBFT v0.38.12 runs inside both MyChain and the ABCI KVStore. MyChain embeds it via Cosmos SDK. The KVStore connects to a standalone CometBFT instance.

### BFT (Byzantine Fault Tolerance)

- **Technical**: The ability of a distributed system to function correctly even if up to 1/3 of participants are malicious or faulty. Named after the Byzantine Generals Problem.
- **Everyday**: A jury that reaches the right verdict even if up to 4 out of 12 jurors are compromised.
- **In this project**: With 4 validators, the chain tolerates 1 malicious validator. With 7 validators, it tolerates 2. This POC uses 1 validator (no fault tolerance -- it is a POC).

### ABCI (Application Blockchain Interface)

- **Technical**: A protocol (socket or gRPC) defining the boundary between the consensus engine (CometBFT) and the application logic. The application implements methods like `CheckTx`, `FinalizeBlock`, `Commit`, and `Query`.
- **Everyday**: A USB port. CometBFT is the computer, your app is the peripheral. ABCI is the USB standard that lets them talk.
- **In this project**: The ABCI KVStore implements ABCI directly (raw interface). The Cosmos SDK implements ABCI internally and provides higher-level abstractions (modules, keepers).

### Proposer

- **Technical**: The validator selected to propose the next block. CometBFT uses a weighted round-robin algorithm based on validator voting power.
- **Everyday**: The person whose turn it is to deal cards in a card game.
- **In this project**: With one validator (`admin`), `admin` proposes every block.

### ABCI Block Lifecycle

The sequence CometBFT calls on your application for each block:

```
PrepareProposal   The proposer selects and orders transactions for the block
        |
ProcessProposal   Other validators verify the proposed block is valid
        |
FinalizeBlock     Execute all transactions and compute state changes
        |
Commit            Persist state changes to disk, return new app hash
```

- **In this project**: The ABCI KVStore implements all four. Cosmos SDK handles this internally for MyChain.

### P2P (Peer-to-Peer)

- **Technical**: The network layer where nodes discover and communicate with each other. CometBFT uses port 26656 for P2P gossip -- sharing transactions and blocks.
- **Everyday**: A group chat where everyone shares news directly with each other, no central server needed.
- **In this project**: Port 26656 (MyChain P2P) and 36656 (KVStore P2P). In a single-node POC, P2P is unused. In production, this is how validators find each other.

### RPC (Remote Procedure Call)

- **Technical**: An HTTP-based interface exposed by CometBFT for external clients to submit transactions, query state, and check node status. Default port 26657.
- **Everyday**: A phone line to the bank. You call in to check your balance or request a transfer.
- **In this project**: The API backend uses RPC (port 26657) to broadcast transactions to MyChain and RPC (port 36657) to interact with the KVStore.

---

## 3. Cosmos SDK

### Cosmos SDK

- **Technical**: A Go framework for building application-specific blockchains. Provides modules (auth, bank, staking, governance, etc.), a module system, and tools for building custom chains on top of CometBFT.
- **Everyday**: A web framework like Django or Rails, but for blockchains. It gives you authentication, a database layer, and common features out of the box.
- **In this project**: MyChain is built with Cosmos SDK v0.50.14. It uses 23 standard + custom modules.

### Module

- **Technical**: A self-contained component in a Cosmos SDK chain that manages a specific domain. Each module has its own state, messages, queries, and logic. Modules interact through keepers.
- **Everyday**: A plugin or extension for a CMS. The "auth" plugin handles login, the "bank" plugin handles money, the "staking" plugin handles deposits.
- **In this project**: `x/taskreg` is a custom module for task management. Standard modules include `auth` (accounts), `bank` (transfers), `staking` (validators), `wasm` (smart contracts).

### Keeper

- **Technical**: The state-access layer for a module. It provides methods to read and write the module's portion of the key-value store. Only the keeper for a module can access that module's state.
- **Everyday**: A librarian who manages one section of the library. Only the science librarian can add or remove science books.
- **In this project**: `TaskregKeeper` has methods like `SetTask()`, `GetTask()`, `GetNextTaskID()`. It stores tasks under the `"Task/value/"` key prefix.

### Message (Msg)

- **Technical**: A typed instruction within a transaction that triggers a state change. Each message has a signer (who authorized it), validation logic, and a handler in the corresponding module.
- **Everyday**: A form you fill out at the bank. "Transfer Form" goes to the transfer department. "Account Opening Form" goes to accounts.
- **In this project**: `MsgCreateTask` (signer: creator), `MsgAssignTask` (signer: creator), `MsgCompleteTask` (signer: assignee).

### Query

- **Technical**: A read-only request that returns data from the current state without creating a transaction. Queries do not go through consensus and are instant.
- **Everyday**: Checking your bank balance online. You are just looking -- not changing anything.
- **In this project**: `QueryTaskRequest{id: 5}` returns task #5. `QueryListTasksRequest{}` returns all tasks. These go through gRPC (port 9090).

### Protobuf (Protocol Buffers)

- **Technical**: A language-neutral, platform-neutral data serialization format by Google. In Cosmos SDK, all messages, queries, and state types are defined in `.proto` files and compiled to Go code.
- **Everyday**: A universal form template. Whether you speak English, Spanish, or Mandarin, you fill in the same fields on the same form.
- **In this project**: Proto files in `mychain/proto/mychain/taskreg/v1/` define Task, MsgCreateTask, QueryTaskRequest, etc. `buf generate` compiles them to Go code.

### gRPC

- **Technical**: A high-performance RPC framework that uses HTTP/2 and Protocol Buffers. Faster and more efficient than REST for service-to-service communication.
- **Everyday**: A high-speed internal phone system between offices, compared to regular mail (REST).
- **In this project**: The API backend queries MyChain via gRPC (port 9090) for task lookups and account info. Much faster than REST for programmatic access.

### Keyring

- **Technical**: A secure storage system for cryptographic keys. Cosmos SDK supports multiple backends: `os` (system keychain), `file` (encrypted file), `test` (plaintext -- dev only).
- **Everyday**: A physical keychain that holds your house key, car key, and office key.
- **In this project**: The API backend uses an in-memory `test` keyring with a fixed mnemonic. It imports one key for signing all transactions. **Never use test keyring in production.**

### Account

- **Technical**: An entity on the blockchain identified by an address, with a sequence number (nonce) and public key. Accounts hold balances and can sign transactions.
- **Everyday**: A bank account with an account number, a balance, and a signature card.
- **In this project**: `admin` and `alice` are accounts created at genesis with 1B stake + 1B token each.

### Address

- **Technical**: A Bech32-encoded string derived from the public key hash. In Cosmos chains, addresses start with a human-readable prefix (e.g., `cosmos1...`).
- **Everyday**: Your bank account number. Derived from your identity but shorter and easier to share.
- **In this project**: Addresses look like `cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za`. The prefix depends on the chain configuration.

### Mnemonic

- **Technical**: A 12 or 24-word phrase (BIP39 standard) that deterministically generates cryptographic keys. The same mnemonic always produces the same keys.
- **Everyday**: A master password that generates all your other passwords. Write it down and keep it safe.
- **In this project**: Admin uses `"abandon abandon ... about"` (12 words). Alice uses `"zoo zoo ... wrong"`. These are test-only mnemonics with known keys. **Never use these in production.**

### HD Path (BIP44)

- **Technical**: A derivation path that determines which key to derive from a mnemonic. Format: `m/purpose'/coin_type'/account'/change/index`. Cosmos uses `m/44'/118'/0'/0/0`.
- **Everyday**: A filing system. The mnemonic is the filing cabinet, and the HD path is "drawer 44, folder 118, page 0, line 0".
- **In this project**: The API backend derives keys at `m/44'/118'/0'/0/0` using the Secp256k1 algorithm.

### Staking / Delegation

- **Technical**: The process of locking tokens to participate in consensus (as a validator) or to support a validator (as a delegator). Staked tokens earn rewards from block production.
- **Everyday**: Putting money in a fixed deposit at a bank. The bank uses it to make loans (secure the network), and you earn interest (staking rewards).
- **In this project**: The `admin` account stakes tokens to become the sole validator via `gentx` during chain initialization.

### Slashing

- **Technical**: A penalty mechanism where a validator loses a portion of staked tokens for misbehavior (double-signing a block or being offline too long).
- **Everyday**: A security deposit that gets partially deducted if you break the rules.
- **In this project**: The `slashing` module is active but irrelevant with one validator. In production, it keeps validators honest.

### IBC (Inter-Blockchain Communication)

- **Technical**: A protocol for trustless communication between independent blockchains. Allows token transfers, data packets, and cross-chain contract calls between Cosmos chains.
- **Everyday**: The postal service between countries. Each country (chain) is independent, but they can send packages (tokens, messages) to each other through a standardized mail system.
- **In this project**: IBC modules (core, transfer, fee) are registered but not actively used. They are ready for connecting to other Cosmos chains.

### Governance (Gov Module)

- **Technical**: An on-chain voting system where token holders propose and vote on changes (parameter updates, software upgrades, fund spending). Proposals pass with sufficient voting power.
- **Everyday**: A shareholder vote at a company meeting. Proposals are submitted, shareholders vote, and the majority wins.
- **In this project**: The `gov` module is active. In production, it would be used for chain upgrades and parameter changes.

---

## 4. CosmWasm

### CosmWasm

- **Technical**: A smart contract platform for the Cosmos ecosystem. Contracts are written in Rust, compiled to WebAssembly (WASM), and executed in a sandboxed virtual machine within the chain.
- **Everyday**: An app store for the blockchain. Developers write apps (contracts) in Rust, upload them, and anyone can use them.
- **In this project**: The Task Bounty contract is a CosmWasm contract that manages financial incentives for tasks.

### WebAssembly (WASM)

- **Technical**: A binary instruction format designed for safe, fast, portable execution. Code compiles to `.wasm` files that run in a virtual machine, isolated from the host system.
- **Everyday**: A universal translator. Write your program once, and it runs identically on any computer.
- **In this project**: The Rust contract compiles to `task_bounty.wasm`. This file is uploaded to MyChain and executed by the `wasmvm` library.

### Smart Contract

- **Technical**: Self-executing code deployed on a blockchain that enforces rules automatically. Once deployed, it runs exactly as programmed -- no one can alter its behavior (unless migration is enabled).
- **Everyday**: A vending machine. Put in money, press a button, get your item. No human middleman needed. The machine enforces the rules.
- **In this project**: The Task Bounty contract automatically calculates fees, splits payments, and prevents double-claiming -- all enforced by code on-chain.

### Contract Lifecycle: Instantiate / Execute / Query

```
Store        Upload WASM bytecode to the chain (gets a Code ID)
    |
Instantiate  Create an instance from the code (gets a Contract Address)
    |
Execute      Call the contract to change state (requires a transaction)
    |
Query        Read contract state (free, no transaction needed)
```

- **Everyday**:
  - Store = Upload an app to the app store
  - Instantiate = Install the app on your phone (with initial settings)
  - Execute = Use the app (tap buttons, change data)
  - Query = View the app screen (read-only)
- **In this project**: `deploy-contracts.sh` stores the WASM, then instantiates with `fee_collector` and `fee_pct=10`.

### Code ID

- **Technical**: A numeric identifier assigned when WASM bytecode is uploaded to the chain. Multiple contract instances can be created from the same Code ID.
- **Everyday**: A product SKU. The same SKU (code) can be used to manufacture many units (instances).
- **In this project**: The task bounty contract gets Code ID `1` (first contract stored). The API backend uses `WASM_CODE_ID=1` to find it.

### Contract Address

- **Technical**: A unique on-chain address assigned to each instantiated contract. Used to send execute messages and queries to that specific instance.
- **Everyday**: The street address of a specific store (instance) that sells products from a catalog (Code ID).
- **In this project**: After instantiation, the contract gets an address like `cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr`. The API backend auto-discovers this from the Code ID.

### wasmd

- **Technical**: The Cosmos SDK module that provides CosmWasm functionality. It handles WASM storage, contract instantiation, execution, and migration. It integrates the `wasmvm` C library for WASM execution.
- **Everyday**: The app runtime on your phone (like Android Runtime). It manages installing, running, and updating apps.
- **In this project**: wasmd v0.53.4 is registered as a module in MyChain's app.go. It provides `MsgStoreCode`, `MsgInstantiateContract`, `MsgExecuteContract`.

### cw-storage-plus

- **Technical**: A Rust library providing typed, efficient storage abstractions for CosmWasm contracts. Offers `Item<T>` (single value) and `Map<K, V>` (key-value collection) with JSON serialization.
- **Everyday**: A typed spreadsheet helper. Instead of raw cells, you get named columns with specific data types.
- **In this project**: `CONFIG: Item<Config>` stores contract configuration. `BOUNTIES: Map<u64, Bounty>` stores all bounties keyed by task ID.

---

## 5. Project-Specific Terms

### Task Registry (x/taskreg)

- **Technical**: A custom Cosmos SDK module in this project that manages task creation, assignment, and completion. Tasks are stored in the chain's key-value store with unique IDs.
- **In this project**: Create tasks, assign them to people, mark them complete -- all on-chain with full auditability.

### Task Status

| Value | Name          | Meaning                                      |
|-------|---------------|----------------------------------------------|
| 0     | UNSPECIFIED   | Default/invalid state                        |
| 1     | OPEN          | Task created, waiting for assignment         |
| 2     | ASSIGNED      | Task assigned to someone, work in progress   |
| 3     | COMPLETED     | Task finished by assignee                    |

State transitions: `OPEN -> ASSIGNED -> COMPLETED` (one-way only).

### Bounty

- **Technical**: A financial reward attached to a task. A funder locks tokens in the contract, and a claimer receives the tokens (minus fees) when they claim the bounty.
- **In this project**: Fund bounty with 1000 tokens for task #1. When someone claims it, they get 900 tokens (at 10% fee), and the fee collector gets 100.

### Funder

- The address that creates and funds a bounty. The funder's tokens are locked in the contract until claimed.

### Claimer

- The address that claims a bounty. They receive the net payout (amount minus fees).

### Fee Collector

- The address that receives platform fees from bounty claims. Set during contract instantiation.

### Multiplier (multiplier_pct)

- A percentage applied to the base bounty amount. Default is 100 (1x). Setting it to 150 means the payout is 1.5x the original amount. Only the admin can change it via `AddBonus`.

### KVStore App

- The standalone ABCI application that provides a simple key-value store backed by Badger DB with its own CometBFT instance. Demonstrates raw ABCI without any framework.

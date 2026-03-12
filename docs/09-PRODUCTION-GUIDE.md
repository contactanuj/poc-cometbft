# Production Deployment Guide

This guide covers everything needed to take this POC to production: multi-validator setup, scaling, monitoring, security, and Kubernetes deployment.

> **Warning**: The POC uses test mnemonics, zero gas fees, a single validator, and no TLS. Every one of these must change for production.

---

## 1. Production Architecture Overview

```
                        +------------------+
                        |  Load Balancer   |
                        |  (TLS termination)|
                        +--------+---------+
                                 |
                    +------------+------------+
                    |                         |
             +------v------+          +------v------+
             | API Backend |          | API Backend |
             | Instance 1  |          | Instance 2  |
             | :8080       |          | :8080       |
             +------+------+          +------+------+
                    |                         |
              gRPC / RPC                 gRPC / RPC
                    |                         |
        +-----------+-----------+-------------+
        |                       |
+-------v--------+     +-------v--------+
| Sentry Node 1  |     | Sentry Node 2  |
| (full node)    |     | (full node)    |
| Public facing  |     | Public facing  |
| :26657 :9090   |     | :26657 :9090   |
+-------+--------+     +-------+--------+
        |                       |
        |    Private Network    |
        |                       |
+-------v--------+     +-------v--------+
| Validator 1    |     | Validator 2    |
| (PRIVATE)      |     | (PRIVATE)      |
| No public ports|     | No public ports|
+----------------+     +-------+--------+
                                |
                        +-------v--------+
                        | Validator 3    |     (Minimum 4 validators for BFT)
                        | (PRIVATE)      |
                        +-------+--------+
                                |
                        +-------v--------+
                        | Validator 4    |
                        | (PRIVATE)      |
                        +----------------+
```

### Why This Topology?

- **Validators are NEVER exposed to the public internet.** They only connect to sentry nodes.
- **Sentry nodes** are public-facing full nodes that relay transactions and blocks between the public internet and validators.
- **If a sentry node is DDoS-attacked**, the validator behind it is unaffected. You can spin up new sentry nodes.
- **API backends** are stateless and horizontally scalable behind a load balancer.

---

## 2. Multi-Validator Setup

### BFT Math

CometBFT tolerates up to **f** faulty validators out of **n** total, where `n >= 3f + 1`:

| Total Validators (n) | Fault Tolerance (f) | Minimum for Consensus |
|-----------------------|---------------------|-----------------------|
| 1                     | 0                   | 1 (no fault tolerance)|
| 4                     | 1                   | 3                     |
| 7                     | 2                   | 5                     |
| 10                    | 3                   | 7                     |
| 21                    | 6                   | 15                    |
| 100                   | 33                  | 67                    |

**Recommendation**: Start with **4 validators** (tolerates 1 failure). Scale to 7+ for higher availability.

### Adding a Validator

```bash
# On the new validator machine:

# 1. Initialize the node
mychaind init validator-2 --chain-id mychain-1

# 2. Copy genesis.json from an existing node
scp existing-node:~/.mychain/config/genesis.json ~/.mychain/config/

# 3. Add persistent peers
# Edit ~/.mychain/config/config.toml
persistent_peers = "node1_id@sentry1:26656,node2_id@sentry2:26656"

# 4. Start the node and let it sync
mychaind start

# 5. Create validator transaction (after node is synced)
mychaind tx staking create-validator \
  --amount 1000000stake \
  --pubkey $(mychaind tendermint show-validator) \
  --moniker "validator-2" \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --from validator2-key \
  --chain-id mychain-1
```

---

## 3. Sentry Node Architecture

```
                    Internet
                       |
          +------------+------------+
          |                         |
   +------v------+          +------v------+
   | Sentry 1    |          | Sentry 2    |
   |             |          |             |
   | pex = true  |          | pex = true  |
   | seed_mode   |          | seed_mode   |
   | = false     |          | = false     |
   |             |          |             |
   | private_    |          | private_    |
   | peer_ids =  |          | peer_ids =  |
   | validator1, |          | validator1, |
   | validator2  |          | validator2  |
   +------+------+          +------+------+
          |                         |
          +--------- LAN ----------+
          |                         |
   +------v------+          +------v------+
   | Validator 1 |          | Validator 2 |
   |             |          |             |
   | pex = false |          | pex = false |
   | persistent_ |          | persistent_ |
   | peers =     |          | peers =     |
   | sentry1,    |          | sentry1,    |
   | sentry2     |          | sentry2     |
   |             |          |             |
   | addr_book_  |          | addr_book_  |
   | strict=false|          | strict=false|
   +-------------+          +-------------+
```

### Sentry Node config.toml

```toml
# Allow peer exchange (discover new peers)
pex = true

# Peers to NEVER gossip about (hide validator identity)
private_peer_ids = "validator1_node_id,validator2_node_id"

# Connect to validators
persistent_peers = "validator1_id@10.0.1.10:26656,validator2_id@10.0.1.11:26656"
```

### Validator config.toml

```toml
# Disable peer exchange (validators don't talk to the public)
pex = false

# Only connect to sentry nodes
persistent_peers = "sentry1_id@10.0.0.10:26656,sentry2_id@10.0.0.11:26656"

# Allow private IPs (for LAN communication)
addr_book_strict = false
```

---

## 4. Hardware Requirements

| Role            | CPU    | RAM   | Disk               | Network      |
|-----------------|--------|-------|--------------------|--------------|
| Validator       | 4+ cores| 16GB+ | 500GB+ NVMe SSD    | 100 Mbps+    |
| Sentry Node     | 4+ cores| 16GB+ | 500GB+ SSD         | 1 Gbps+      |
| API Backend     | 2+ cores| 4GB+  | 50GB SSD           | 100 Mbps+    |
| Full Node       | 2+ cores| 8GB+  | 500GB+ SSD         | 100 Mbps+    |

**Disk**: State grows over time. Plan for ~1-5 GB/month depending on transaction volume. Enable pruning to limit growth.

---

## 5. CometBFT Tuning for High Load

### config.toml -- Consensus

```toml
[consensus]
# Block time (lower = faster but more network load)
timeout_propose = "3s"        # Default: 3s
timeout_prevote = "1s"        # Default: 1s
timeout_precommit = "1s"      # Default: 1s
timeout_commit = "5s"         # Default: 5s (time between blocks)

# For high throughput, reduce timeout_commit:
# timeout_commit = "1s"       # ~1 block/second (high load on validators)
```

### config.toml -- Mempool

```toml
[mempool]
# Maximum transactions in mempool
size = 5000                   # Default: 5000

# Maximum bytes of transactions in mempool
max_txs_bytes = 1073741824    # Default: 1GB

# Maximum size of a single transaction
max_tx_bytes = 1048576        # Default: 1MB

# Cache of already-seen transactions (prevent resubmission)
cache_size = 10000            # Default: 10000
```

### config.toml -- P2P

```toml
[p2p]
# Maximum peer connections
max_num_inbound_peers = 40    # Default: 40
max_num_outbound_peers = 10   # Default: 10

# Peer exchange
pex = true                    # Disable on validators
```

---

## 6. Cosmos SDK Production Configuration

### app.toml -- Minimum Gas Prices

```toml
# CRITICAL: Set non-zero gas prices to prevent spam
minimum-gas-prices = "0.025stake"
```

### app.toml -- Pruning

```toml
# Pruning strategy
pruning = "custom"
pruning-keep-recent = "100"      # Keep last 100 blocks
pruning-interval = "10"          # Run pruning every 10 blocks

# Alternatives:
# pruning = "nothing"    # Keep all state (archive node, huge disk)
# pruning = "everything" # Keep only current state (minimal disk)
# pruning = "default"    # Keep 362880 blocks (~21 days at 5s blocks)
```

### app.toml -- State Sync

```toml
[state-sync]
# Enable state sync snapshots for new nodes to fast-sync
snapshot-interval = 1000         # Create snapshot every 1000 blocks
snapshot-keep-recent = 2         # Keep last 2 snapshots
```

### app.toml -- Telemetry

```toml
[telemetry]
enabled = true
service-name = "mychain"
prometheus-retention-time = 60   # Seconds to retain metrics

# Prometheus will scrape from :26660/metrics (CometBFT)
# and the application telemetry endpoint
```

### app.toml -- gRPC

```toml
[grpc]
enable = true
address = "0.0.0.0:9090"
max-recv-msg-size = 10485760     # 10MB max message size
max-send-msg-size = 10485760
```

---

## 7. API Backend Scaling

### Horizontal Scaling

```
                 +------------------+
                 |  Load Balancer   |
                 |  (Nginx/HAProxy) |
                 +--------+---------+
                          |
            +-------------+-------------+
            |             |             |
     +------v---+  +------v---+  +------v---+
     | Backend 1|  | Backend 2|  | Backend 3|
     | Key: A   |  | Key: B   |  | Key: C   |
     | Seq: 100 |  | Seq: 200 |  | Seq: 300 |
     +----------+  +----------+  +----------+
            |             |             |
            +------+------+------+------+
                   |             |
            +------v---+  +------v---+
            | Node 1   |  | Node 2   |
            | (Sentry) |  | (Sentry) |
            +----------+  +----------+
```

### Removing the Mutex Bottleneck

The POC's single-account mutex is the biggest throughput limiter. Production solutions:

**Option A: Multiple Signing Accounts**
- Create N accounts (e.g., 10)
- Each API backend instance uses a different account
- No mutex needed between instances
- Each instance still needs internal mutex for its own account

**Option B: Account Sequence Queuing**
- Predict the next sequence number
- Increment locally instead of querying the chain each time
- Queue transactions and batch-submit
- Requires careful error handling for sequence mismatches

**Option C: Shared Transaction Queue**
- Use Redis/NATS as a transaction queue
- A single dedicated signer service reads from the queue and broadcasts
- API backends only push to the queue (fire-and-forget)
- Highest throughput, more complex architecture

### Key Management (CRITICAL)

| POC (NEVER in production)        | Production                              |
|-----------------------------------|-----------------------------------------|
| Test mnemonic in env variable     | HSM (Hardware Security Module)          |
| In-memory keyring                 | Vault / AWS KMS / Azure Key Vault       |
| Same key for all operations       | Separate keys per service / operation   |
| Plaintext in docker-compose.yml   | Kubernetes Secrets / sealed secrets     |

---

## 8. CosmWasm Production Considerations

### Contract Migration

```bash
# Deploy with admin (allows future upgrades)
mychaind tx wasm instantiate 1 "$INIT_MSG" \
  --label "task-bounty-v1" \
  --admin cosmos1governance... \   # Use governance address
  --from deployer

# Later, migrate to new code
mychaind tx wasm migrate $CONTRACT_ADDR 2 "$MIGRATE_MSG" \
  --from governance-key
```

**Best practice**: Set admin to the governance module address. Contract upgrades require a governance vote.

### Gas Limits

```toml
# In app.toml
[wasm]
# Maximum gas for smart contract execution
simulation_gas_limit = 50000000      # 50M gas max per query
memory_cache_size = 512              # MB of WASM module cache

# Maximum WASM code size
max_wasm_code_size = 1228800         # 1.2MB
```

### Contract Auditing Checklist

- [ ] No integer overflow in payout calculations
- [ ] No unauthorized admin functions
- [ ] Reentrancy protection (CosmWasm handles this by default)
- [ ] Correct fund handling (all sent funds accounted for)
- [ ] Pagination limits enforced
- [ ] Error messages do not leak sensitive data

---

## 9. Monitoring and Observability

### Monitoring Stack

```
+------------------------------------------------------------------+
|                        Monitoring Stack                            |
|                                                                   |
|  +-------------+    +-----------+    +-----------+               |
|  | CometBFT    |--->| Prometheus|--->| Grafana   |               |
|  | :26660      |    | :9090     |    | :3000     |               |
|  | /metrics    |    |           |    | Dashboards|               |
|  +-------------+    |           |    +-----------+               |
|                     |           |                                 |
|  +-------------+    |           |    +-----------+               |
|  | API Backend |--->|           |--->| AlertMgr  |               |
|  | /metrics    |    |           |    | :9093     |               |
|  +-------------+    +-----------+    +-----------+               |
|                                            |                      |
|  +-------------+                     +-----v-----+               |
|  | Node        |                     | PagerDuty |               |
|  | Exporter    |                     | Slack     |               |
|  | (system)    |                     | Email     |               |
|  +-------------+                     +-----------+               |
+------------------------------------------------------------------+
```

### Key Metrics to Monitor

| Metric                                    | Alert Threshold          | Meaning                           |
|-------------------------------------------|--------------------------|-----------------------------------|
| `cometbft_consensus_height`               | No increase for 30s      | Chain stopped producing blocks    |
| `cometbft_consensus_validators`           | Drops below expected     | Validator went offline            |
| `cometbft_consensus_missing_validators`   | > 0 for 5 min            | Validator missing rounds          |
| `cometbft_p2p_peers`                      | < 2                      | Node isolated from network        |
| `cometbft_mempool_size`                   | > 1000                   | Mempool backlog (chain too slow)  |
| `cometbft_consensus_block_interval`       | > 10s average            | Block time degraded               |
| `disk_usage_percent`                      | > 80%                    | Running out of disk space         |
| `process_resident_memory_bytes`           | > 80% of RAM             | Memory pressure                   |
| `http_request_duration_seconds` (API)     | p99 > 30s                | API timeouts                      |

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'cometbft'
    static_configs:
      - targets: ['mychain-node:26660']
    metrics_path: /metrics

  - job_name: 'api-backend'
    static_configs:
      - targets: ['api-backend:8080']
    metrics_path: /metrics

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

### Log Aggregation

Use structured logging with ELK Stack or Grafana Loki:

```yaml
# docker-compose.production.yml
services:
  mychain-node:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
        tag: "mychain-node"
```

---

## 10. Backup and Recovery

### State Snapshots

```bash
# Enable snapshots in app.toml
[state-sync]
snapshot-interval = 1000
snapshot-keep-recent = 2

# List available snapshots
mychaind snapshots list

# Export genesis (full state dump)
mychaind export > genesis_export.json
```

### Recovery Procedure

```bash
# Option 1: State sync from existing nodes (fastest)
mychaind start --state-sync.snapshot-interval=0

# Option 2: Restore from genesis export
mychaind init new-node --chain-id mychain-1
cp genesis_export.json ~/.mychain/config/genesis.json
mychaind start

# Option 3: Replay from genesis (slowest, most complete)
# Start with original genesis.json and let the node sync all blocks
```

### Backup Schedule

| What                | How Often    | Retention  |
|---------------------|--------------|------------|
| Validator key       | Once (secure)| Forever    |
| Chain data snapshot | Daily        | 7 days     |
| Genesis export      | Weekly       | 30 days    |
| Config files        | On change    | In git     |

---

## 11. Security Hardening Checklist

### Network Security

- [ ] Validators behind sentry nodes (never public)
- [ ] Firewall: only P2P port open on sentry nodes
- [ ] Firewall: RPC/gRPC only accessible from API backends
- [ ] TLS on all public endpoints (API, RPC, gRPC)
- [ ] Rate limiting on public RPC/API endpoints
- [ ] DDoS protection (Cloudflare, AWS Shield) on sentry nodes

### Key Security

- [ ] Validator keys stored in HSM (YubiHSM, AWS CloudHSM)
- [ ] No test mnemonics (generate fresh keys)
- [ ] Key backup encrypted and stored offline
- [ ] Separate keys for validator signing and transaction signing
- [ ] Key rotation procedure documented and tested

### Application Security

- [ ] Non-zero gas prices (prevent spam)
- [ ] Rate limiting on API endpoints
- [ ] Input validation on all endpoints
- [ ] CORS restricted to known origins
- [ ] API authentication (JWT/API keys) for write operations
- [ ] Audit logs for all state-changing operations

### Operational Security

- [ ] SSH key-based access only (no passwords)
- [ ] Principle of least privilege for all service accounts
- [ ] Secrets managed via Vault/KMS (not env vars or files)
- [ ] Automated security updates for OS and dependencies
- [ ] Incident response runbook documented

---

## 12. Kubernetes Deployment

### Architecture

```
+------------------------------------------------------------------+
|  Kubernetes Cluster                                               |
|                                                                   |
|  +------------------+   +------------------+                     |
|  | StatefulSet:     |   | StatefulSet:     |                     |
|  | validator        |   | sentry           |                     |
|  | replicas: 4      |   | replicas: 3      |                     |
|  |                  |   |                  |                     |
|  | PVC: 500Gi SSD   |   | PVC: 500Gi SSD   |                     |
|  +------------------+   +------------------+                     |
|                                                                   |
|  +------------------+   +------------------+                     |
|  | Deployment:      |   | Service:         |                     |
|  | api-backend      |   | api-lb           |                     |
|  | replicas: 3      |   | type: LoadBalancer|                    |
|  |                  |   | port: 443 -> 8080|                     |
|  | HPA: 3-10 pods   |   +------------------+                     |
|  +------------------+                                             |
|                                                                   |
|  +------------------+   +------------------+                     |
|  | ConfigMap:       |   | Secret:          |                     |
|  | chain-config     |   | validator-keys   |                     |
|  | (config.toml,    |   | (priv_validator  |                     |
|  |  app.toml)       |   |  _key.json)      |                     |
|  +------------------+   +------------------+                     |
+------------------------------------------------------------------+
```

### StatefulSet for Validators

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: validator
spec:
  serviceName: validator
  replicas: 4
  selector:
    matchLabels:
      app: validator
  template:
    metadata:
      labels:
        app: validator
    spec:
      containers:
      - name: mychaind
        image: mychain:latest
        ports:
        - containerPort: 26656
          name: p2p
        - containerPort: 26657
          name: rpc
        - containerPort: 9090
          name: grpc
        resources:
          requests:
            cpu: "4"
            memory: "16Gi"
          limits:
            cpu: "8"
            memory: "32Gi"
        livenessProbe:
          httpGet:
            path: /status
            port: 26657
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /status
            port: 26657
          initialDelaySeconds: 10
          periodSeconds: 5
        volumeMounts:
        - name: chain-data
          mountPath: /root/.mychain
  volumeClaimTemplates:
  - metadata:
      name: chain-data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: ssd
      resources:
        requests:
          storage: 500Gi
```

### HPA for API Backend

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-backend
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-backend
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## 13. Capacity Planning

### Transaction Throughput Estimation

| Block Time | Txs per Block | TPS (theoretical) |
|------------|---------------|--------------------|
| 5s         | 100           | 20                 |
| 5s         | 1000          | 200                |
| 1s         | 100           | 100                |
| 1s         | 1000          | 1000               |

**Factors that affect throughput:**
- Block size (max_bytes in genesis.json)
- Transaction size (simple task vs complex contract execution)
- Validator count (more validators = slower consensus)
- Network latency between validators
- Hardware (CPU for signature verification, disk I/O for state writes)

### State Growth

| Data Type           | Estimated Size per Record | 1M Records |
|---------------------|--------------------------|------------|
| Task (protobuf)     | ~200 bytes               | ~200 MB    |
| Bounty (JSON)       | ~300 bytes               | ~300 MB    |
| KV pair             | variable                 | variable   |
| Block metadata      | ~2 KB per block          | ~12 GB/yr at 5s blocks |

### Scaling Triggers

| Metric                     | Action                                      |
|----------------------------|---------------------------------------------|
| TPS approaching max        | Reduce block time or increase block size    |
| Disk > 70% full            | Enable aggressive pruning or expand disk    |
| API latency p99 > 5s       | Add more API backend instances              |
| Missed blocks > 0          | Check validator hardware / network          |
| Mempool consistently full  | Increase gas prices to filter low-value txs |

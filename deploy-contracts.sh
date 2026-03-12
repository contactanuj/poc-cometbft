#!/bin/bash
set -e

CHAIN_RPC="${CHAIN_RPC:-http://localhost:26657}"
CHAIN_REST="${CHAIN_REST:-http://localhost:1317}"
CHAIN_ID="${CHAIN_ID:-mychain-1}"
KEYRING="test"
HOME_DIR="/root/.mychain"
WASM_FILE="${WASM_FILE:-/contracts/artifacts/task_bounty.wasm}"

echo "=== Deploy CosmWasm Contract ==="

# Wait for chain to be ready
echo "Waiting for chain..."
for i in $(seq 1 60); do
    if curl -s "$CHAIN_RPC/status" | grep -q '"catching_up":false'; then
        echo "Chain is ready!"
        break
    fi
    echo "Waiting... ($i/60)"
    sleep 2
done

# Get admin address
ADMIN_ADDR=$(mychaind keys show admin -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "Admin address: $ADMIN_ADDR"

# Store the wasm contract
echo "Storing contract..."
TX_RESULT=$(mychaind tx wasm store "$WASM_FILE" \
    --from admin \
    --keyring-backend "$KEYRING" \
    --chain-id "$CHAIN_ID" \
    --gas auto \
    --gas-adjustment 1.5 \
    --fees 0stake \
    --home "$HOME_DIR" \
    --output json \
    --yes 2>&1)

echo "Store TX result: $TX_RESULT"
sleep 6  # Wait for block

# Get the code ID (should be 1 for first deployment)
CODE_ID=1
echo "Code ID: $CODE_ID"

# Instantiate the contract
echo "Instantiating contract..."
INIT_MSG=$(cat <<EOF
{
    "fee_collector": "$ADMIN_ADDR",
    "fee_pct": 10
}
EOF
)

TX_RESULT=$(mychaind tx wasm instantiate "$CODE_ID" "$INIT_MSG" \
    --from admin \
    --keyring-backend "$KEYRING" \
    --chain-id "$CHAIN_ID" \
    --label "task-bounty" \
    --admin "$ADMIN_ADDR" \
    --gas auto \
    --gas-adjustment 1.5 \
    --fees 0stake \
    --home "$HOME_DIR" \
    --output json \
    --yes 2>&1)

echo "Instantiate TX result: $TX_RESULT"
sleep 6  # Wait for block

# Get contract address
CONTRACT_ADDR=$(mychaind query wasm list-contract-by-code "$CODE_ID" \
    --home "$HOME_DIR" \
    --output json | jq -r '.contracts[0]')

echo ""
echo "=== Deployment Complete ==="
echo "Code ID: $CODE_ID"
echo "Contract Address: $CONTRACT_ADDR"
echo ""

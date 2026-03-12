#!/bin/bash
set -e

CMTHOME="${CMTHOME:-/data/cometbft}"

# Initialize CometBFT if not already done
if [ ! -f "$CMTHOME/config/genesis.json" ]; then
    echo "Initializing CometBFT..."
    cometbft init --home "$CMTHOME"

    # Configure proxy_app
    sed -i "s|proxy_app = \"tcp://127.0.0.1:26658\"|proxy_app = \"tcp://127.0.0.1:36658\"|g" "$CMTHOME/config/config.toml"
    # Configure RPC listen address
    sed -i "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:36657\"|g" "$CMTHOME/config/config.toml"
    # Configure P2P listen address
    sed -i "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:36656\"|g" "$CMTHOME/config/config.toml"
fi

# Start the ABCI app in background
echo "Starting kvstore ABCI app..."
kvstore-app --addr tcp://0.0.0.0:36658 --db /data/kvstore-db &

# Wait a moment for ABCI to start
sleep 2

# Start CometBFT in foreground
echo "Starting CometBFT..."
exec cometbft node --home "$CMTHOME"

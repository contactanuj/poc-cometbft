#!/bin/bash
set -e

CHAIN_ID="${CHAIN_ID:-mychain-1}"
MONIKER="${MONIKER:-mychain-node}"
KEYRING="test"
HOME_DIR="${HOME_DIR:-/root/.mychain}"

# Fixed test mnemonics for deterministic addresses
ADMIN_MNEMONIC="abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
ALICE_MNEMONIC="zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong"

# Only initialize if not already done
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
    echo "Initializing chain..."

    mychaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

    # Create keys from fixed mnemonics
    echo "Creating keys..."
    echo "$ADMIN_MNEMONIC" | mychaind keys add admin --keyring-backend "$KEYRING" --home "$HOME_DIR" --recover
    echo "$ALICE_MNEMONIC" | mychaind keys add alice --keyring-backend "$KEYRING" --home "$HOME_DIR" --recover

    ADMIN_ADDR=$(mychaind keys show admin -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
    ALICE_ADDR=$(mychaind keys show alice -a --keyring-backend "$KEYRING" --home "$HOME_DIR")

    echo "Admin address: $ADMIN_ADDR"
    echo "Alice address: $ALICE_ADDR"

    # Add genesis accounts
    mychaind add-genesis-account "$ADMIN_ADDR" 1000000000stake,1000000000token --home "$HOME_DIR"
    mychaind add-genesis-account "$ALICE_ADDR" 1000000000stake,1000000000token --home "$HOME_DIR"

    # Create gentx
    mychaind gentx admin 100000000stake --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING" --home "$HOME_DIR"
    mychaind collect-gentxs --home "$HOME_DIR"
    mychaind validate-genesis --home "$HOME_DIR"

    # Enable API
    sed -i 's/enable = false/enable = true/g' "$HOME_DIR/config/app.toml"
    sed -i 's/swagger = false/swagger = true/g' "$HOME_DIR/config/app.toml"
    sed -i 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' "$HOME_DIR/config/app.toml"

    # Set minimum gas prices
    sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0stake"/g' "$HOME_DIR/config/app.toml"

    # Enable gRPC and bind to all interfaces
    sed -i '/\[grpc\]/,/^\[/ s/enable = false/enable = true/' "$HOME_DIR/config/app.toml"
    sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|g' "$HOME_DIR/config/app.toml"

    # Configure RPC to listen on all interfaces
    sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$HOME_DIR/config/config.toml"

    # Allow CORS
    sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/g' "$HOME_DIR/config/config.toml"

    echo "Chain initialized successfully!"
fi

echo "Starting chain..."
exec mychaind start --home "$HOME_DIR"

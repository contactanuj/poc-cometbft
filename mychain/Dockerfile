FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git make build-base linux-headers wget

WORKDIR /app

# Download wasmvm library
RUN ARCH=$(uname -m) && \
    wget -q https://github.com/CosmWasm/wasmvm/releases/download/v2.1.4/libwasmvm_muslc.${ARCH}.a \
    -O /lib/libwasmvm_muslc.a && \
    cp /lib/libwasmvm_muslc.a /lib/libwasmvm.${ARCH}.a && \
    cp /lib/libwasmvm_muslc.a /lib/libwasmvm_muslc.${ARCH}.a

COPY go.mod ./
RUN go mod download 2>/dev/null || true

COPY . .

RUN go mod tidy && CGO_ENABLED=1 BUILD_TAGS="netgo,ledger,muslc" \
    go build -tags "$BUILD_TAGS" \
    -ldflags="-w -s -extldflags '-Wl,-z,muldefs -static -Wl,-z,stacksize=8388608'" \
    -trimpath \
    -o /mychaind ./cmd/mychaind

# Runtime stage
FROM alpine:3.18

RUN apk add --no-cache bash sed jq curl

COPY --from=builder /mychaind /usr/local/bin/mychaind
COPY scripts/init.sh /scripts/init.sh
RUN chmod +x /scripts/init.sh

EXPOSE 26656 26657 1317 9090

CMD ["/scripts/init.sh"]

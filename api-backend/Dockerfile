FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git make build-base linux-headers wget

WORKDIR /workspace

# Download wasmvm for CGO (needed by mychain dependency)
RUN ARCH=$(uname -m) && \
    wget -q https://github.com/CosmWasm/wasmvm/releases/download/v2.1.4/libwasmvm_muslc.${ARCH}.a \
    -O /lib/libwasmvm_muslc.a && \
    cp /lib/libwasmvm_muslc.a /lib/libwasmvm.${ARCH}.a && \
    cp /lib/libwasmvm_muslc.a /lib/libwasmvm_muslc.${ARCH}.a

# Copy all source (go.work links the modules)
COPY go.work ./
COPY mychain/ ./mychain/
COPY api-backend/ ./api-backend/

# Tidy and download deps
RUN cd mychain && go mod tidy && cd ../api-backend && go mod tidy

RUN cd api-backend && \
    CGO_ENABLED=1 go build -tags "netgo,ledger,muslc" \
    -ldflags="-w -s -extldflags '-Wl,-z,muldefs -static -Wl,-z,stacksize=8388608'" \
    -trimpath \
    -o /api-backend .

# Runtime
FROM alpine:3.18
RUN apk add --no-cache bash curl
COPY --from=builder /api-backend /usr/local/bin/api-backend
EXPOSE 8080
CMD ["api-backend"]

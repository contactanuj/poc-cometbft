# Build stage
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /kvstore-app .

# Install CometBFT
FROM golang:1.22-alpine AS cometbft-builder
RUN apk add --no-cache git make
RUN go install github.com/cometbft/cometbft/cmd/cometbft@v0.38.12

# Runtime
FROM alpine:3.18
RUN apk add --no-cache bash curl
COPY --from=builder /kvstore-app /usr/local/bin/kvstore-app
COPY --from=cometbft-builder /go/bin/cometbft /usr/local/bin/cometbft
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 36656 36657 36658
ENTRYPOINT ["/entrypoint.sh"]

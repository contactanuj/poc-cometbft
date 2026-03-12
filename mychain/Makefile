.PHONY: build install proto-gen clean

BUILD_DIR := ./build
BINARY := mychaind

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/mychaind

install:
	go install ./cmd/mychaind

proto-gen:
	@echo "Generating protobuf files..."
	docker run --rm -v $(CURDIR):/workspace --workdir /workspace \
		ghcr.io/cosmos/proto-builder:0.14.0 \
		sh -c "cd proto && buf generate --template buf.gen.gogo.yaml"

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...

lint:
	golangci-lint run ./...

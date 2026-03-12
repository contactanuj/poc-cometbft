package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	myapp "poc-cometbft/mychain/app"

	"poc-cometbft/api-backend/client"
	"poc-cometbft/api-backend/handlers"
)

func main() {
	chainGRPC := envOrDefault("CHAIN_GRPC_ADDR", "localhost:9090")
	chainRPC := envOrDefault("CHAIN_RPC_ADDR", "http://localhost:26657")
	kvstoreRPC := envOrDefault("KVSTORE_RPC_ADDR", "http://localhost:36657")
	chainID := envOrDefault("CHAIN_ID", "mychain-1")
	mnemonic := envOrDefault("TEST_MNEMONIC", "")
	wasmCodeIDStr := envOrDefault("WASM_CODE_ID", "1")
	port := envOrDefault("PORT", "8080")

	wasmCodeID, _ := strconv.ParseUint(wasmCodeIDStr, 10, 64)

	log.Printf("[main] Starting API backend with config: chainGRPC=%s chainRPC=%s kvstoreRPC=%s chainID=%s wasmCodeID=%d port=%s",
		chainGRPC, chainRPC, kvstoreRPC, chainID, wasmCodeID, port)

	// Wait for chain readiness
	log.Println("Waiting for chain to be ready...")
	waitForReady(chainRPC+"/status", 60)
	log.Println("Chain is ready!")

	// Wait for kvstore readiness
	log.Println("Waiting for kvstore to be ready...")
	waitForReady(kvstoreRPC+"/status", 60)
	log.Println("KVStore is ready!")

	// Initialize encoding
	encCfg := myapp.MakeEncodingConfig()

	// Create chain client
	chainClient, err := client.NewChainClient(
		chainGRPC, chainRPC, chainID, mnemonic,
		encCfg.Codec, encCfg.InterfaceRegistry, encCfg.TxConfig,
	)
	if err != nil {
		log.Fatalf("Failed to create chain client: %v", err)
	}

	// Resolve contract address by code ID (non-blocking, contract may not be deployed yet)
	if wasmCodeID > 0 {
		log.Printf("Resolving contract for code ID %d...", wasmCodeID)
		for i := 0; i < 5; i++ {
			addr, err := chainClient.ResolveContractByCodeID(context.Background(), wasmCodeID)
			if err == nil {
				chainClient.SetContractAddr(addr)
				log.Printf("Contract address: %s", addr)
				break
			}
			if i == 4 {
				log.Printf("Contract not deployed yet (code ID %d). Bounty endpoints will be unavailable. Deploy with deploy-contracts.sh", wasmCodeID)
			} else {
				log.Printf("Contract not found yet, retrying... (%v)", err)
				time.Sleep(3 * time.Second)
			}
		}
	}

	// Create kvstore client
	kvClient, err := client.NewKVStoreClient(kvstoreRPC)
	if err != nil {
		log.Fatalf("Failed to create kvstore client: %v", err)
	}

	// Setup handlers
	taskHandler := handlers.NewTaskHandler(chainClient)
	bountyHandler := handlers.NewBountyHandler(chainClient)
	kvHandler := handlers.NewKVStoreHandler(kvClient)
	healthHandler := handlers.NewHealthHandler(chainClient, kvClient)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health
	r.Get("/api/health", healthHandler.Health)

	// Tasks (SDK module)
	r.Post("/api/tasks", taskHandler.CreateTask)
	r.Get("/api/tasks", taskHandler.ListTasks)
	r.Get("/api/tasks/{id}", taskHandler.GetTask)
	r.Post("/api/tasks/{id}/assign", taskHandler.AssignTask)
	r.Post("/api/tasks/{id}/complete", taskHandler.CompleteTask)

	// Bounties (CosmWasm)
	r.Post("/api/bounties/fund", bountyHandler.FundBounty)
	r.Post("/api/bounties/claim", bountyHandler.ClaimBounty)
	r.Get("/api/bounties/{id}", bountyHandler.GetBounty)
	r.Get("/api/bounties", bountyHandler.ListBounties)
	r.Get("/api/config", bountyHandler.GetConfig)

	// KV Store (ABCI)
	r.Put("/api/kv/{key}", kvHandler.SetKey)
	r.Get("/api/kv/{key}", kvHandler.GetKey)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("API backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func waitForReady(url string, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	log.Printf("Warning: service at %s not ready after %d retries", url, maxRetries)
}

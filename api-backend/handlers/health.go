package handlers

import (
	"log"
	"net/http"

	"poc-cometbft/api-backend/client"
)

type HealthHandler struct {
	chainClient   *client.ChainClient
	kvstoreClient *client.KVStoreClient
}

func NewHealthHandler(cc *client.ChainClient, kv *client.KVStoreClient) *HealthHandler {
	return &HealthHandler{chainClient: cc, kvstoreClient: kv}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] GET /api/health: checking health")
	status := map[string]string{}

	// Check chain
	if err := h.chainClient.CheckHealth(r.Context()); err != nil {
		log.Printf("[api] GET /api/health: chain error=%v", err)
		status["chain"] = "error: " + err.Error()
	} else {
		status["chain"] = "ok"
	}

	// Check kvstore
	if err := h.kvstoreClient.CheckHealth(r.Context()); err != nil {
		log.Printf("[api] GET /api/health: kvstore error=%v", err)
		status["kvstore"] = "error: " + err.Error()
	} else {
		status["kvstore"] = "ok"
	}

	log.Printf("[api] GET /api/health: chain=%s kvstore=%s", status["chain"], status["kvstore"])
	respondJSON(w, http.StatusOK, status)
}

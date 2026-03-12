package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"poc-cometbft/api-backend/client"
)

type KVStoreHandler struct {
	client *client.KVStoreClient
}

func NewKVStoreHandler(c *client.KVStoreClient) *KVStoreHandler {
	return &KVStoreHandler{client: c}
}

type SetKVRequest struct {
	Value string `json:"value"`
}

func (h *KVStoreHandler) SetKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		log.Printf("[api] PUT /api/kv: missing key")
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	var req SetKVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[api] PUT /api/kv/%s: bad request err=%v", key, err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("[api] PUT /api/kv/%s: valueLen=%d", key, len(req.Value))
	result, err := h.client.Set(r.Context(), key, req.Value)
	if err != nil {
		log.Printf("[api] PUT /api/kv/%s: error=%v", key, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] PUT /api/kv/%s: success txHash=%s", key, result["tx_hash"])
	respondJSON(w, http.StatusOK, result)
}

func (h *KVStoreHandler) GetKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		log.Printf("[api] GET /api/kv: missing key")
		respondError(w, http.StatusBadRequest, "key is required")
		return
	}

	log.Printf("[api] GET /api/kv/%s: fetching value", key)
	value, err := h.client.Get(r.Context(), key)
	if err != nil {
		log.Printf("[api] GET /api/kv/%s: error=%v", key, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/kv/%s: found valueLen=%d", key, len(value))
	respondJSON(w, http.StatusOK, map[string]string{
		"key":   key,
		"value": value,
	})
}

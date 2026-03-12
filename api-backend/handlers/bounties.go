package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"poc-cometbft/api-backend/client"
)

type BountyHandler struct {
	client *client.ChainClient
}

func NewBountyHandler(c *client.ChainClient) *BountyHandler {
	return &BountyHandler{client: c}
}

type FundBountyRequest struct {
	TaskID uint64 `json:"task_id"`
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

type ClaimBountyRequest struct {
	TaskID uint64 `json:"task_id"`
}

func (h *BountyHandler) FundBounty(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] POST /api/bounties/fund: decoding request")
	var req FundBountyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[api] POST /api/bounties/fund: bad request err=%v", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("[api] POST /api/bounties/fund: taskID=%d amount=%s denom=%s", req.TaskID, req.Amount, req.Denom)
	result, err := h.client.FundBounty(r.Context(), req.TaskID, req.Amount, req.Denom)
	if err != nil {
		log.Printf("[api] POST /api/bounties/fund: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] POST /api/bounties/fund: success txHash=%v", result["tx_hash"])
	respondJSON(w, http.StatusCreated, result)
}

func (h *BountyHandler) ClaimBounty(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] POST /api/bounties/claim: decoding request")
	var req ClaimBountyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[api] POST /api/bounties/claim: bad request err=%v", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("[api] POST /api/bounties/claim: taskID=%d", req.TaskID)
	result, err := h.client.ClaimBounty(r.Context(), req.TaskID)
	if err != nil {
		log.Printf("[api] POST /api/bounties/claim: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] POST /api/bounties/claim: success txHash=%v", result["tx_hash"])
	respondJSON(w, http.StatusOK, result)
}

func (h *BountyHandler) GetBounty(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Printf("[api] GET /api/bounties/%s: invalid bounty id", idStr)
		respondError(w, http.StatusBadRequest, "invalid bounty id")
		return
	}

	log.Printf("[api] GET /api/bounties/%d: fetching bounty", id)
	bounty, err := h.client.GetBounty(r.Context(), id)
	if err != nil {
		log.Printf("[api] GET /api/bounties/%d: error=%v", id, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/bounties/%d: found", id)
	respondJSON(w, http.StatusOK, bounty)
}

func (h *BountyHandler) ListBounties(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] GET /api/bounties: listing all bounties")
	bounties, err := h.client.ListBounties(r.Context())
	if err != nil {
		log.Printf("[api] GET /api/bounties: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/bounties: success")
	respondJSON(w, http.StatusOK, bounties)
}

func (h *BountyHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] GET /api/config: fetching contract config")
	config, err := h.client.GetContractConfig(r.Context())
	if err != nil {
		log.Printf("[api] GET /api/config: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/config: success")
	respondJSON(w, http.StatusOK, config)
}

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"poc-cometbft/api-backend/client"
)

type TaskHandler struct {
	client *client.ChainClient
}

func NewTaskHandler(c *client.ChainClient) *TaskHandler {
	return &TaskHandler{client: c}
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AssignTaskRequest struct {
	Assignee string `json:"assignee"`
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] POST /api/tasks: decoding request")
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[api] POST /api/tasks: bad request err=%v", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		log.Printf("[api] POST /api/tasks: missing title")
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	log.Printf("[api] POST /api/tasks: title=%q", req.Title)
	result, err := h.client.CreateTask(r.Context(), req.Title, req.Description)
	if err != nil {
		log.Printf("[api] POST /api/tasks: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] POST /api/tasks: created successfully txHash=%v", result["tx_hash"])
	respondJSON(w, http.StatusCreated, result)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	log.Printf("[api] GET /api/tasks: listing all tasks")
	tasks, err := h.client.ListTasks(r.Context())
	if err != nil {
		log.Printf("[api] GET /api/tasks: error=%v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/tasks: success")
	respondJSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Printf("[api] GET /api/tasks/%s: invalid task id", idStr)
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	log.Printf("[api] GET /api/tasks/%d: fetching task", id)
	task, err := h.client.GetTask(r.Context(), id)
	if err != nil {
		log.Printf("[api] GET /api/tasks/%d: error=%v", id, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] GET /api/tasks/%d: found", id)
	respondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Printf("[api] POST /api/tasks/%s/assign: invalid task id", idStr)
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var req AssignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[api] POST /api/tasks/%d/assign: bad request err=%v", id, err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("[api] POST /api/tasks/%d/assign: assignee=%q", id, req.Assignee)
	result, err := h.client.AssignTask(r.Context(), id, req.Assignee)
	if err != nil {
		log.Printf("[api] POST /api/tasks/%d/assign: error=%v", id, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] POST /api/tasks/%d/assign: success txHash=%v", id, result["tx_hash"])
	respondJSON(w, http.StatusOK, result)
}

func (h *TaskHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Printf("[api] POST /api/tasks/%s/complete: invalid task id", idStr)
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	log.Printf("[api] POST /api/tasks/%d/complete: completing task", id)
	result, err := h.client.CompleteTask(r.Context(), id)
	if err != nil {
		log.Printf("[api] POST /api/tasks/%d/complete: error=%v", id, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[api] POST /api/tasks/%d/complete: success txHash=%v", id, result["tx_hash"])
	respondJSON(w, http.StatusOK, result)
}

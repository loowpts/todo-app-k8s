package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"todo-app/internal/repository"
	"todo-app/internal/service"
)

type TaskAPIHandler struct {
	svc *service.TaskService
}

func NewTaskAPIHandler(svc *service.TaskService) *TaskAPIHandler {
	return &TaskAPIHandler{svc: svc}
}

type taskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

func (h *TaskAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *TaskAPIHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	task, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskAPIHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, errors.New("title is required"))
		return
	}

	task, err := h.svc.Create(r.Context(), req.Title, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskAPIHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, errors.New("title is required"))
		return
	}

	task, err := h.svc.Update(r.Context(), id, req.Title, req.Description, req.Done)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskAPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := idFromURL(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.Delete(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func idFromURL(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

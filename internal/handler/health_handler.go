package handler

import (
	"net/http"
	"time"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, r, http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

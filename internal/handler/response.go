package handler

import (
	"net/http"

	"github.com/go-chi/render"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func RespondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	render.Status(r, status)
	render.JSON(w, r, APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

func RespondError(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    status,
			Message: message,
		},
	})
}

package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"DumbProtocol/internal/service"
)

type TOTPHandler struct {
	totpService service.TOTPService
}

func NewTOTPHandler(totpService service.TOTPService) *TOTPHandler {
	return &TOTPHandler{
		totpService: totpService,
	}
}

type SetupRequest struct {
	AccountName string `json:"account_name"`
	Issuer      string `json:"issuer,omitempty"`
}

type VerifyRequest struct {
	AccountName string `json:"account_name"`
	Code        string `json:"code"`
}

type RecoveryRequest struct {
	AccountName string `json:"account_name"`
	Action      string `json:"action,omitempty"` // "generate" or "verify"
	Code        string `json:"code,omitempty"`   // required if action is "verify"
	Count       int    `json:"count,omitempty"`  // optional code count for "generate"
}

// Setup handles POST /api/v1/totp/setup
func (h *TOTPHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.AccountName == "" {
		RespondError(w, r, http.StatusBadRequest, "account_name is required")
		return
	}

	result, err := h.totpService.Setup(r.Context(), req.AccountName, req.Issuer)
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, r, http.StatusOK, result)
}

// Verify handles POST /api/v1/totp/verify
func (h *TOTPHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.AccountName == "" {
		RespondError(w, r, http.StatusBadRequest, "account_name is required")
		return
	}
	if req.Code == "" {
		RespondError(w, r, http.StatusBadRequest, "code is required")
		return
	}

	result, err := h.totpService.Verify(r.Context(), req.AccountName, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrSecretNotFound) {
			RespondError(w, r, http.StatusNotFound, err.Error())
			return
		}
		RespondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if !result.Valid {
		RespondJSON(w, r, http.StatusUnauthorized, result)
		return
	}

	RespondJSON(w, r, http.StatusOK, result)
}

// Recovery handles POST /api/v1/totp/recovery
func (h *TOTPHandler) Recovery(w http.ResponseWriter, r *http.Request) {
	var req RecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.AccountName == "" {
		RespondError(w, r, http.StatusBadRequest, "account_name is required")
		return
	}

	switch req.Action {
	case "verify":
		if req.Code == "" {
			RespondError(w, r, http.StatusBadRequest, "code is required for backup code verification")
			return
		}
		res, err := h.totpService.VerifyBackupCode(r.Context(), req.AccountName, req.Code)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if !res.Valid {
			RespondJSON(w, r, http.StatusUnauthorized, res)
			return
		}
		RespondJSON(w, r, http.StatusOK, res)

	case "generate", "":
		count := req.Count
		if count <= 0 {
			count = 8
		}
		codes, err := h.totpService.GenerateBackupCodes(r.Context(), req.AccountName, count)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		RespondJSON(w, r, http.StatusOK, map[string]interface{}{
			"account_name": req.AccountName,
			"backup_codes": codes,
			"message":      "New backup codes generated. Previous codes have been invalidated.",
		})

	default:
		RespondError(w, r, http.StatusBadRequest, "unsupported action: must be 'generate' or 'verify'")
	}
}

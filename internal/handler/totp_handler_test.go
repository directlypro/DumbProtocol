package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"DumbProtocol/internal/config"
	"DumbProtocol/internal/database"
	"DumbProtocol/internal/handler"
	"DumbProtocol/internal/repository"
	"DumbProtocol/internal/service"
)

func TestTOTPHTTPEndpoints(t *testing.T) {
	db, err := database.NewConnection("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	repo := repository.NewTOTPRepository(db)
	totpService := service.NewTOTPService(repo, "DumbProtocol")
	cfg := &config.Config{
		Env:         "test",
		Host:        "127.0.0.1",
		Port:        8080,
		ReadTimeout: 5 * time.Second,
	}

	srv, err := handler.NewServer(cfg, totpService)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	testServer := httptest.NewServer(srv.RouterHandler())
	defer testServer.Close()

	// 1. Setup Endpoint: POST /api/v1/totp/setup
	setupReqBody, _ := json.Marshal(map[string]string{
		"account_name": "bob@example.com",
		"issuer":       "DumbProtocol",
	})
	resp, err := http.Post(testServer.URL+"/api/v1/totp/setup", "application/json", bytes.NewBuffer(setupReqBody))
	if err != nil {
		t.Fatalf("POST /api/v1/totp/setup failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/v1/totp/setup, got %d", resp.StatusCode)
	}

	// 2. Current Code Endpoint: POST /api/v1/totp/code
	codeReqBody, _ := json.Marshal(map[string]string{
		"account_name": "bob@example.com",
	})
	resp, err = http.Post(testServer.URL+"/api/v1/totp/code", "application/json", bytes.NewBuffer(codeReqBody))
	if err != nil {
		t.Fatalf("POST /api/v1/totp/code failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/v1/totp/code, got %d", resp.StatusCode)
	}

	var codeEnvelope handler.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&codeEnvelope); err != nil {
		t.Fatalf("failed to decode code response: %v", err)
	}
	if !codeEnvelope.Success {
		t.Fatalf("expected success true in code response")
	}
}

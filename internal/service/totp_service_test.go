package service_test

import (
	"context"
	"testing"
	"time"

	"DumbProtocol/internal/database"
	"DumbProtocol/internal/repository"
	"DumbProtocol/internal/service"

	"github.com/pquerna/otp/totp"
)

func setupTestDB(t *testing.T) *database.DB {
	db, err := database.NewConnection("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}
	return db
}

func TestTOTPServiceFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := repository.NewTOTPRepository(db)
	totpSvc := service.NewTOTPService(repo, "TestApp")
	ctx := context.Background()

	account := "alice@example.com"

	// 1. Test Setup
	setupRes, err := totpSvc.Setup(ctx, account, "TestApp")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if setupRes.AccountName != account {
		t.Errorf("expected account %s, got %s", account, setupRes.AccountName)
	}
	if setupRes.Secret == "" {
		t.Error("expected non-empty secret")
	}

	// 2. Test GetCurrentCode (SMS Gateway requirement)
	codeRes, err := totpSvc.GetCurrentCode(ctx, account)
	if err != nil {
		t.Fatalf("GetCurrentCode failed: %v", err)
	}
	if len(codeRes.Code) != 6 {
		t.Errorf("expected 6-digit code, got %s", codeRes.Code)
	}
	if codeRes.ExpiresInSeconds <= 0 || codeRes.ExpiresInSeconds > 30 {
		t.Errorf("invalid expiresInSeconds: %d", codeRes.ExpiresInSeconds)
	}

	// 3. Test Verification with valid TOTP code
	validCode, err := totp.GenerateCode(setupRes.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to generate test TOTP code: %v", err)
	}

	verifyRes, err := totpSvc.Verify(ctx, account, validCode)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !verifyRes.Valid {
		t.Errorf("expected valid TOTP verification, got invalid")
	}
}

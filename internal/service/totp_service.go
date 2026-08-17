package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"DumbProtocol/internal/repository"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

var (
	ErrAccountRequired = errors.New("account_name is required")
	ErrSecretNotFound  = errors.New("TOTP secret not found for account")
	ErrInvalidCode     = errors.New("invalid TOTP code")
	ErrInvalidBackup   = errors.New("invalid or already used backup code")
)

type TOTPSetupResult struct {
	AccountName string   `json:"account_name"`
	Issuer      string   `json:"issuer"`
	Secret      string   `json:"secret"`
	URL         string   `json:"otpauth_url"`
	QRCodeData  string   `json:"qr_code"` // data:image/png;base64,...
	BackupCodes []string `json:"backup_codes,omitempty"`
}

type VerificationResult struct {
	AccountName string `json:"account_name"`
	Valid       bool   `json:"valid"`
	Method      string `json:"method"` // "totp" or "backup_code"
	Message     string `json:"message"`
}

type TOTPService interface {
	Setup(ctx context.Context, accountName, issuer string) (*TOTPSetupResult, error)
	Verify(ctx context.Context, accountName, code string) (*VerificationResult, error)
	GenerateBackupCodes(ctx context.Context, accountName string, count int) ([]string, error)
	VerifyBackupCode(ctx context.Context, accountName, code string) (*VerificationResult, error)
}

type totpService struct {
	repo   repository.TOTPRepository
	issuer string
}

func NewTOTPService(repo repository.TOTPRepository, defaultIssuer string) TOTPService {
	if defaultIssuer == "" {
		defaultIssuer = "DumbProtocol"
	}
	return &totpService{
		repo:   repo,
		issuer: defaultIssuer,
	}
}

func (s *totpService) Setup(ctx context.Context, accountName, issuer string) (*TOTPSetupResult, error) {
	if accountName == "" {
		return nil, ErrAccountRequired
	}

	if issuer == "" {
		issuer = s.issuer
	}

	// Generate new TOTP Key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate QR Code PNG bytes
	pngBytes, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	qrCodeDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)

	// Save to DB
	rec := &repository.TOTPSecretRecord{
		AccountName: accountName,
		Secret:      key.Secret(),
		Algorithm:   "SHA1",
		Digits:      6,
		Period:      30,
	}

	if err := s.repo.SaveTOTPSecret(ctx, rec); err != nil {
		return nil, fmt.Errorf("failed to save TOTP secret: %w", err)
	}

	// Generate initial set of backup codes
	backupCodes, err := s.GenerateBackupCodes(ctx, accountName, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	return &TOTPSetupResult{
		AccountName: accountName,
		Issuer:      issuer,
		Secret:      key.Secret(),
		URL:         key.URL(),
		QRCodeData:  qrCodeDataURI,
		BackupCodes: backupCodes,
	}, nil
}

func (s *totpService) Verify(ctx context.Context, accountName, code string) (*VerificationResult, error) {
	if accountName == "" {
		return nil, ErrAccountRequired
	}
	if code == "" {
		return nil, ErrInvalidCode
	}

	rec, err := s.repo.GetTOTPSecret(ctx, accountName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to retrieve TOTP secret: %w", err)
	}

	// Validate TOTP with 1 period skew allowance (clock drift window +/- 30s)
	opts := totp.ValidateOpts{
		Period:    uint(rec.Period),
		Skew:      1,
		Digits:    6,
		Algorithm: 0, // SHA1
	}

	valid, err := totp.ValidateCustom(code, rec.Secret, time.Now().UTC(), opts)
	if err != nil || !valid {
		// Fallback check: check if user provided a backup code instead
		backupRes, backupErr := s.VerifyBackupCode(ctx, accountName, code)
		if backupErr == nil && backupRes.Valid {
			return backupRes, nil
		}

		return &VerificationResult{
			AccountName: accountName,
			Valid:       false,
			Method:      "totp",
			Message:     "Invalid authentication code",
		}, nil
	}

	return &VerificationResult{
		AccountName: accountName,
		Valid:       true,
		Method:      "totp",
		Message:     "TOTP code validated successfully",
	}, nil
}

func (s *totpService) GenerateBackupCodes(ctx context.Context, accountName string, count int) ([]string, error) {
	if accountName == "" {
		return nil, ErrAccountRequired
	}
	if count <= 0 {
		count = 8
	}

	plainCodes := make([]string, count)
	codeHashes := make([]string, count)

	for i := 0; i < count; i++ {
		code, hash, err := generateRandomBackupCode()
		if err != nil {
			return nil, err
		}
		plainCodes[i] = code
		codeHashes[i] = hash
	}

	if err := s.repo.SaveBackupCodes(ctx, accountName, codeHashes); err != nil {
		return nil, fmt.Errorf("failed to save backup codes: %w", err)
	}

	return plainCodes, nil
}

func (s *totpService) VerifyBackupCode(ctx context.Context, accountName, code string) (*VerificationResult, error) {
	if accountName == "" {
		return nil, ErrAccountRequired
	}
	if code == "" {
		return nil, ErrInvalidBackup
	}

	hash := hashBackupCode(code)
	used, err := s.repo.MarkBackupCodeAsUsed(ctx, accountName, hash)
	if err != nil {
		return nil, fmt.Errorf("error verifying backup code: %w", err)
	}

	if !used {
		return &VerificationResult{
			AccountName: accountName,
			Valid:       false,
			Method:      "backup_code",
			Message:     "Invalid or previously redeemed backup code",
		}, nil
	}

	return &VerificationResult{
		AccountName: accountName,
		Valid:       true,
		Method:      "backup_code",
		Message:     "Backup recovery code accepted and marked as used",
	}, nil
}

func generateRandomBackupCode() (code string, hash string, err error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	raw := hex.EncodeToString(bytes) // 10 chars
	// format as 5-5 split e.g. "a1b2c-3d4e5"
	code = fmt.Sprintf("%s-%s", raw[:5], raw[5:])
	hash = hashBackupCode(code)
	return code, hash, nil
}

func hashBackupCode(code string) string {
	// Simple SHA-256 hashing for backup codes
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

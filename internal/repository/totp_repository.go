package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"DumbProtocol/internal/database"
)

var (
	ErrNotFound = errors.New("record not found")
)

type TOTPSecretRecord struct {
	ID          int64
	AccountName string
	Secret      string
	Algorithm   string
	Digits      int
	Period      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BackupCodeRecord struct {
	ID          int64
	AccountName string
	CodeHash    string
	Used        bool
	CreatedAt   time.Time
	UsedAt      *time.Time
}

type TOTPRepository interface {
	SaveTOTPSecret(ctx context.Context, record *TOTPSecretRecord) error
	GetTOTPSecret(ctx context.Context, accountName string) (*TOTPSecretRecord, error)
	DeleteTOTPSecret(ctx context.Context, accountName string) error
	SaveBackupCodes(ctx context.Context, accountName string, codeHashes []string) error
	GetBackupCodes(ctx context.Context, accountName string) ([]BackupCodeRecord, error)
	MarkBackupCodeAsUsed(ctx context.Context, accountName, codeHash string) (bool, error)
}

type sqlTOTPRepository struct {
	db *database.DB
}

func NewTOTPRepository(db *database.DB) TOTPRepository {
	return &sqlTOTPRepository{db: db}
}

func (r *sqlTOTPRepository) SaveTOTPSecret(ctx context.Context, rec *TOTPSecretRecord) error {
	now := time.Now().UTC()
	rec.UpdatedAt = now

	// Check if existing record exists
	existing, err := r.GetTOTPSecret(ctx, rec.AccountName)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if existing != nil {
		query := `UPDATE totp_secrets SET secret = ?, algorithm = ?, digits = ?, period = ?, updated_at = ? WHERE account_name = ?`
		if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
			query = `UPDATE totp_secrets SET secret = $1, algorithm = $2, digits = $3, period = $4, updated_at = $5 WHERE account_name = $6`
		}
		_, err := r.db.ExecContext(ctx, query, rec.Secret, rec.Algorithm, rec.Digits, rec.Period, now, rec.AccountName)
		return err
	}

	rec.CreatedAt = now
	query := `INSERT INTO totp_secrets (account_name, secret, algorithm, digits, period, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		query = `INSERT INTO totp_secrets (account_name, secret, algorithm, digits, period, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}

	res, err := r.db.ExecContext(ctx, query, rec.AccountName, rec.Secret, rec.Algorithm, rec.Digits, rec.Period, rec.CreatedAt, rec.UpdatedAt)
	if err != nil {
		return err
	}
	if id, err := res.LastInsertId(); err == nil {
		rec.ID = id
	}
	return nil
}

func (r *sqlTOTPRepository) GetTOTPSecret(ctx context.Context, accountName string) (*TOTPSecretRecord, error) {
	query := `SELECT id, account_name, secret, algorithm, digits, period, created_at, updated_at FROM totp_secrets WHERE account_name = ?`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		query = `SELECT id, account_name, secret, algorithm, digits, period, created_at, updated_at FROM totp_secrets WHERE account_name = $1`
	}

	row := r.db.QueryRowContext(ctx, query, accountName)
	var rec TOTPSecretRecord
	err := row.Scan(&rec.ID, &rec.AccountName, &rec.Secret, &rec.Algorithm, &rec.Digits, &rec.Period, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (r *sqlTOTPRepository) DeleteTOTPSecret(ctx context.Context, accountName string) error {
	query := `DELETE FROM totp_secrets WHERE account_name = ?`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		query = `DELETE FROM totp_secrets WHERE account_name = $1`
	}
	_, err := r.db.ExecContext(ctx, query, accountName)
	return err
}

func (r *sqlTOTPRepository) SaveBackupCodes(ctx context.Context, accountName string, codeHashes []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete old backup codes for this account
	delQuery := `DELETE FROM backup_codes WHERE account_name = ?`
	insQuery := `INSERT INTO backup_codes (account_name, code_hash, used, created_at) VALUES (?, ?, 0, ?)`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		delQuery = `DELETE FROM backup_codes WHERE account_name = $1`
		insQuery = `INSERT INTO backup_codes (account_name, code_hash, used, created_at) VALUES ($1, $2, FALSE, $3)`
	}

	if _, err := tx.ExecContext(ctx, delQuery, accountName); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, codeHash := range codeHashes {
		if _, err := tx.ExecContext(ctx, insQuery, accountName, codeHash, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *sqlTOTPRepository) GetBackupCodes(ctx context.Context, accountName string) ([]BackupCodeRecord, error) {
	query := `SELECT id, account_name, code_hash, used, created_at, used_at FROM backup_codes WHERE account_name = ?`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		query = `SELECT id, account_name, code_hash, used, created_at, used_at FROM backup_codes WHERE account_name = $1`
	}

	rows, err := r.db.QueryContext(ctx, query, accountName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []BackupCodeRecord
	for rows.Next() {
		var rec BackupCodeRecord
		var usedInt int
		var usedAt sql.NullTime
		if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
			if err := rows.Scan(&rec.ID, &rec.AccountName, &rec.CodeHash, &rec.Used, &rec.CreatedAt, &usedAt); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&rec.ID, &rec.AccountName, &rec.CodeHash, &usedInt, &rec.CreatedAt, &usedAt); err != nil {
				return nil, err
			}
			rec.Used = usedInt != 0
		}
		if usedAt.Valid {
			rec.UsedAt = &usedAt.Time
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *sqlTOTPRepository) MarkBackupCodeAsUsed(ctx context.Context, accountName, codeHash string) (bool, error) {
	now := time.Now().UTC()
	query := `UPDATE backup_codes SET used = 1, used_at = ? WHERE account_name = ? AND code_hash = ? AND used = 0`
	if r.db.Driver == "postgres" || r.db.Driver == "postgresql" {
		query = `UPDATE backup_codes SET used = TRUE, used_at = $1 WHERE account_name = $2 AND code_hash = $3 AND used = FALSE`
	}

	res, err := r.db.ExecContext(ctx, query, now, accountName, codeHash)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

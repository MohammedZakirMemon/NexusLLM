package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type APIKey struct {
	ID        string
	TenantID  string
	KeyPrefix string
	Tier      string
	RPMLimit  int
}

type APIKeyStore struct {
	db *sql.DB
}

func NewAPIKeyStore(db *sql.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

// Generate creates a new raw API key, hashes it, stores the hash, returns the raw key once.
func (s *APIKeyStore) Generate(ctx context.Context, tenantID, tier string, rpmLimit int) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawKey := "nxl_" + hex.EncodeToString(raw)
	prefix := rawKey[:12]

	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO api_keys (tenant_id, key_hash, key_prefix, tier, rpm_limit) VALUES ($1, $2, $3, $4, $5)`,
		tenantID, string(hash), prefix, tier, rpmLimit,
	)
	if err != nil {
		return "", fmt.Errorf("store api key: %w", err)
	}
	return rawKey, nil
}

// Validate checks the raw key against stored hashes and returns the key metadata.
func (s *APIKeyStore) Validate(ctx context.Context, rawKey string) (*APIKey, error) {
	if len(rawKey) < 12 {
		return nil, fmt.Errorf("invalid key format")
	}
	prefix := rawKey[:12]

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tenant_id, key_hash, tier, rpm_limit FROM api_keys WHERE key_prefix = $1 AND is_active = TRUE`,
		prefix,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k APIKey
		var hash string
		if err := rows.Scan(&k.ID, &k.TenantID, &hash, &k.Tier, &k.RPMLimit); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(rawKey)) == nil {
			s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`, time.Now(), k.ID)
			k.KeyPrefix = prefix
			return &k, nil
		}
	}
	return nil, fmt.Errorf("invalid api key")
}

package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Connect(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) Migrate() error {
	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
			key_hash TEXT NOT NULL UNIQUE,
			key_prefix TEXT NOT NULL,
			tier TEXT NOT NULL DEFAULT 'free',
			rpm_limit INT NOT NULL DEFAULT 60,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_used_at TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS usage_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID REFERENCES tenants(id),
			api_key_id UUID REFERENCES api_keys(id),
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			prompt_tokens INT,
			completion_tokens INT,
			total_tokens INT,
			latency_ms INT,
			cache_hit BOOLEAN DEFAULT FALSE,
			status_code INT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_usage_logs_tenant_id ON usage_logs(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
	`)
	return err
}

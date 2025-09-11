package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type PostgresAgentInfoStore struct {
	db *sql.DB
}

func NewPostgresAgentInfoStore(db *sql.DB) *PostgresAgentInfoStore {
	return &PostgresAgentInfoStore{db: db}
}

// EnsureSchema creates the required tables and indexes if they do not exist.
// Safe to call multiple times.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS agent_info (
  agent_id      TEXT PRIMARY KEY,
  provider_id   TEXT NOT NULL,
  agent_type_id TEXT NOT NULL,
  name          TEXT NOT NULL,
  service_type_id  TEXT NOT NULL,
  service_group_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_info_provider_id   ON agent_info (provider_id);
CREATE INDEX IF NOT EXISTS idx_agent_info_agent_type_id ON agent_info (agent_type_id);
`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}

func (s *PostgresAgentInfoStore) Create(ctx context.Context, a AgentInfo) error {
	const q = `
INSERT INTO agent_info (agent_id, provider_id, agent_type_id, name, service_type_id, service_group_id)
VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.db.ExecContext(ctx, q, a.AgentId, a.ProviderId, a.AgentTypeId, a.Name, a.ServiceTypeId, a.ServiceGroupId)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("create agent_info: %w", err)
	}
	return nil
}

func (s *PostgresAgentInfoStore) Upsert(ctx context.Context, a AgentInfo) error {
	const q = `
INSERT INTO agent_info (agent_id, provider_id, agent_type_id, name, service_type_id, service_group_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (agent_id) DO UPDATE
SET provider_id = EXCLUDED.provider_id,
    agent_type_id = EXCLUDED.agent_type_id,
    name = EXCLUDED.name`
	_, err := s.db.ExecContext(ctx, q, a.AgentId, a.ProviderId, a.AgentTypeId, a.Name, a.ServiceTypeId, a.ServiceGroupId)
	if err != nil {
		return fmt.Errorf("upsert agent_info: %w", err)
	}
	return nil
}

func (s *PostgresAgentInfoStore) GetByName(ctx context.Context, agentName string) (AgentInfo, error) {
	const q = `
SELECT agent_id, provider_id, agent_type_id, name, service_type_id, service_group_id
FROM agent_info
WHERE name = $1`
	var a AgentInfo
	err := s.db.QueryRowContext(ctx, q, agentName).Scan(
		&a.AgentId, &a.ProviderId, &a.AgentTypeId, &a.Name, &a.ServiceTypeId, &a.ServiceGroupId,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentInfo{}, ErrNotFound
		}
		return AgentInfo{}, fmt.Errorf("get agent_info: %w", err)
	}
	return a, nil
}

// isUniqueViolation detects Postgres unique violations without binding to a specific driver error type.
// If you use pgx or lib/pq, you can tighten this to check SQLState "23505".
func isUniqueViolation(err error) bool {
	// Fallback: string contains check to avoid importing driver-specific packages.
	// Replace with pgx/libpq SQLState check if you standardize on a driver.
	msg := err.Error()
	return contains(msg, "duplicate key value") || contains(msg, "unique constraint")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}

// Minimal indexOf to avoid extra imports.
func indexOf(s, sub string) int {
	n := len(sub)
	if n == 0 {
		return 0
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return i
		}
	}
	return -1
}

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
  token_id      TEXT NOT NULL,
  name          TEXT NOT NULL
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
INSERT INTO agent_info (agent_id, provider_id, agent_type_id, token_id, name)
VALUES ($1, $2, $3, $4, $5)`
	_, err := s.db.ExecContext(ctx, q, a.AgentId, a.ProviderId, a.AgentTypeId, a.TokenId, a.Name)
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
INSERT INTO agent_info (agent_id, provider_id, agent_type_id, token_id, name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (agent_id) DO UPDATE
SET provider_id = EXCLUDED.provider_id,
    agent_type_id = EXCLUDED.agent_type_id,
    token_id = EXCLUDED.token_id,
    name = EXCLUDED.name`
	_, err := s.db.ExecContext(ctx, q, a.AgentId, a.ProviderId, a.AgentTypeId, a.TokenId, a.Name)
	if err != nil {
		return fmt.Errorf("upsert agent_info: %w", err)
	}
	return nil
}

func (s *PostgresAgentInfoStore) GetByID(ctx context.Context, agentID string) (AgentInfo, error) {
	const q = `
SELECT agent_id, provider_id, agent_type_id, token_id, name
FROM agent_info
WHERE agent_id = $1`
	var a AgentInfo
	err := s.db.QueryRowContext(ctx, q, agentID).Scan(
		&a.AgentId, &a.ProviderId, &a.AgentTypeId, &a.TokenId, &a.Name,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentInfo{}, ErrNotFound
		}
		return AgentInfo{}, fmt.Errorf("get agent_info: %w", err)
	}
	return a, nil
}

func (s *PostgresAgentInfoStore) GetByName(ctx context.Context, agentName string) (AgentInfo, error) {
	const q = `
SELECT agent_id, provider_id, agent_type_id, token_id, name
FROM agent_info
WHERE name = $1`
	var a AgentInfo
	err := s.db.QueryRowContext(ctx, q, agentName).Scan(
		&a.AgentId, &a.ProviderId, &a.AgentTypeId, &a.TokenId, &a.Name,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentInfo{}, ErrNotFound
		}
		return AgentInfo{}, fmt.Errorf("get agent_info: %w", err)
	}
	return a, nil
}

func (s *PostgresAgentInfoStore) List(ctx context.Context, filter ListFilter) ([]AgentInfo, error) {
	base := `
SELECT agent_id, provider_id, agent_type_id, token_id, name
FROM agent_info`
	where := ""
	args := []any{}
	i := 1

	if filter.ProviderId != "" {
		where += fmt.Sprintf(" provider_id = $%d", i)
		args = append(args, filter.ProviderId)
		i++
	}
	if filter.AgentTypeId != "" {
		if where != "" {
			where += " AND"
		}
		where += fmt.Sprintf(" agent_type_id = $%d", i)
		args = append(args, filter.AgentTypeId)
		i++
	}
	q := base
	if where != "" {
		q += " WHERE" + where
	}
	q += " ORDER BY agent_id"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent_info: %w", err)
	}
	defer rows.Close()

	var out []AgentInfo
	for rows.Next() {
		var a AgentInfo
		if err := rows.Scan(&a.AgentId, &a.ProviderId, &a.AgentTypeId, &a.TokenId, &a.Name); err != nil {
			return nil, fmt.Errorf("scan agent_info: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent_info: %w", err)
	}
	return out, nil
}

func (s *PostgresAgentInfoStore) Delete(ctx context.Context, agentID string) error {
	const q = `DELETE FROM agent_info WHERE agent_id = $1`
	res, err := s.db.ExecContext(ctx, q, agentID)
	if err != nil {
		return fmt.Errorf("delete agent_info: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
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

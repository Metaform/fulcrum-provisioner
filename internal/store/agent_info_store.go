package store

import "context"

type AgentInfo struct {
	AgentId     string
	ProviderId  string
	AgentTypeId string
	TokenId     string
	Name        string
}

type AgentInfoStore interface {
	// Create inserts a new AgentInfo. Returns ErrAlreadyExists if AgentId exists.
	Create(ctx context.Context, a AgentInfo) error
	// Upsert inserts or updates an AgentInfo by AgentId.
	Upsert(ctx context.Context, a AgentInfo) error
	// GetByID fetches an AgentInfo by AgentId. Returns ErrNotFound if missing.
	GetByID(ctx context.Context, agentID string) (AgentInfo, error)
	// GetByName fetches an AgentInfo by AgentName. Returns ErrNotFound if missing.
	GetByName(ctx context.Context, agentName string) (AgentInfo, error)
	// List returns AgentInfo records, optionally filtered by ProviderId or AgentTypeId.
	List(ctx context.Context, filter ListFilter) ([]AgentInfo, error)
	// Delete removes an AgentInfo by AgentId. Returns ErrNotFound if missing.
	Delete(ctx context.Context, agentID string) error
}

type ListFilter struct {
	ProviderId  string
	AgentTypeId string
}

var (
	ErrNotFound      = errNotFound("agent not found")
	ErrAlreadyExists = errAlreadyExists("agent already exists")
)

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

type errAlreadyExists string

func (e errAlreadyExists) Error() string { return string(e) }

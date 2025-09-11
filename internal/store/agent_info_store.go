package store

import "context"

type AgentInfo struct {
	AgentId        string
	ProviderId     string
	AgentTypeId    string
	Name           string
	ServiceTypeId  string
	ServiceGroupId string
}

type AgentInfoStore interface {
	// Create inserts a new AgentInfo. Returns ErrAlreadyExists if AgentId exists.
	Create(ctx context.Context, a AgentInfo) error
	// Upsert inserts or updates an AgentInfo by AgentId.
	Upsert(ctx context.Context, a AgentInfo) error
	// GetByName fetches an AgentInfo by AgentName. Returns ErrNotFound if missing.
	GetByName(ctx context.Context, agentName string) (AgentInfo, error)
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

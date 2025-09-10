package model

import "time"

type ParticipantDefinition struct {
	ParticipantName       string `json:"participantName,omitempty" validate:"required"`
	Did                   string `json:"did,omitempty" validate:"required"`
	KubernetesIngressHost string `json:"kubeHost,omitempty"`
}
type PendingJob struct {
	Id         string                 `json:"id"`
	ProviderId string                 `json:"providerId"`
	ConsumerId string                 `json:"consumerId"`
	AgentId    string                 `json:"agentId"`
	ServiceId  string                 `json:"serviceId"`
	Action     string                 `json:"action"`
	Params     map[string]interface{} `json:"params"`
	Status     string                 `json:"status"`
	Priority   int                    `json:"priority"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Service    struct {
		Id            string                 `json:"id"`
		ProviderId    string                 `json:"providerId"`
		ConsumerId    string                 `json:"consumerId"`
		AgentId       string                 `json:"agentId"`
		ServiceTypeId string                 `json:"serviceTypeId"`
		GroupId       string                 `json:"groupId"`
		Name          string                 `json:"name"`
		Status        string                 `json:"status"`
		Properties    map[string]interface{} `json:"properties"`
		CreatedAt     time.Time              `json:"createdAt"`
		UpdatedAt     time.Time              `json:"updatedAt"`
	} `json:"service"`
}

type ParticipantData struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AgentData struct {
	Name          string                 `json:"name"`
	ProviderId    string                 `json:"providerId"`
	AgentTypeId   string                 `json:"agentTypeId"`
	Tags          []string               `json:"tags"`
	Configuration map[string]interface{} `json:"configuration"`
	Participant   ParticipantData        `json:"participant,omitempty"`
}

type TokenInformation struct {
	Id            string    `json:"id"`
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	ExpireAt      time.Time `json:"expireAt"`
	ParticipantId string    `json:"participantId"`
	AgentId       string    `json:"agentId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ListTokenResponse struct {
	Items []TokenInformation `json:"items"`
}

type TokenData struct {
	Id            string    `json:"id"`
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	ExpireAt      time.Time `json:"expireAt"`
	ScopeId       string    `json:"scopeId"`
	AgentId       string    `json:"agentId"`
	ParticipantId string    `json:"participantId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Value         string    `json:"value"`
}

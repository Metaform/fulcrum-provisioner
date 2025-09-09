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

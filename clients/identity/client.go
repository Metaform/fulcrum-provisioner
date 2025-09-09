package clients

import (
	"k8s-provisioner/clients/config"

	"k8s.io/apimachinery/pkg/util/json"
)

type IdentityApi interface {
	CreateParticipant(body string) (string, error)
}

type IdentityApiClient struct {
	config.ApiConfig
}

type ParticipantResponse struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	ApiKey       string `json:"apiKey"`
}

func (i *IdentityApiClient) CreateParticipant(body string) (*ParticipantResponse, error) {
	jsonBody, err := config.SendRequest(i.HttpClient, i.ApiKey, body, i.BaseUrl+"/participants")

	if err != nil {
		return nil, err
	}

	var p ParticipantResponse
	err = json.Unmarshal([]byte(jsonBody), &p)
	if err != nil {
		return nil, nil
	}
	return &p, err
}

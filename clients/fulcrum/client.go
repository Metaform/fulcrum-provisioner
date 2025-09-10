package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"k8s-provisioner/clients/config"
	"k8s-provisioner/internal/model"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FulcrumApi interface {

	// seeding functions, that should be invoked sequentially as soon as the provisioner comes up
	CreateServiceType(id string, name string) (string, error)
	CreateAgentType(serviceTypeId string, name string) (string, error)
	CreateParticipant(name string) (string, error)
	CreateServiceGroup(providerId string, name string) (string, error)
	CreateAgent(agentData model.AgentData) (string, error)
	CreateAgentToken(agentId string, tokenName string) (string, error)
	ListTokens() ([]model.TokenInformation, error)
	RegenerateToken(tokenId string) (*model.TokenData, error)
	// these functions are invoked by the provisioner to get and process jobs
	GetPendingJobs(agentToken string) ([]model.PendingJob, error)
	ClaimJob(agentToken string, jobId string) error
	FinalizeJob(agentToken string, jobId string) error
}

type FulcrumApiClient struct {
	config.ApiConfig
}

func NewFulcrumApiClient(baseUrl string) FulcrumApi {
	return &FulcrumApiClient{
		ApiConfig: config.ApiConfig{
			ApiKey:     "change-me",
			HttpClient: config.CreateHttpClient(),
			BaseUrl:    baseUrl,
		},
	}
}

type IdResponse struct {
	Id string `json:"id"`
}

func (f *FulcrumApiClient) CreateServiceType(id string, name string) (string, error) {

	body := `{
		"id": "` + id + `",
		"name": "` + name + `"
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/service-types", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := IdResponse{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func (f *FulcrumApiClient) CreateAgentType(serviceTypeId string, name string) (string, error) {
	body := `{
		"serviceTypeIds": ["` + serviceTypeId + `"],
		"name": "` + name + `"
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/agent-types", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := IdResponse{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func (f *FulcrumApiClient) CreateParticipant(name string) (string, error) {
	body := `{
		"status": "Enabled",
		"name": "` + name + `"
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/participants", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := IdResponse{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func (f *FulcrumApiClient) CreateServiceGroup(providerId string, name string) (string, error) {
	body := `{
		"consumerId": "` + providerId + `",
		"name": "` + name + `"
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/service-groups", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := IdResponse{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func (f *FulcrumApiClient) CreateAgent(agentData model.AgentData) (string, error) {

	body, err := json.Marshal(agentData)
	if err != nil {
		return "", err
	}
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/agents", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := IdResponse{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Id, nil
}

func (f *FulcrumApiClient) CreateAgentToken(agentId string, tokenName string) (string, error) {
	body := `{
		"scopeId": "` + agentId + `",
		"name": "` + tokenName + `",
		"role": "agent",
        "expireAt": "` + time.Now().Add(time.Hour*24*365).Format(time.RFC3339) + `"
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/tokens", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	response, err := f.requestWithResponse(rq)
	if err != nil {
		return "", err
	}
	r := model.TokenData{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return "", err
	}
	return r.Value, nil
}

func (f *FulcrumApiClient) ListTokens() ([]model.TokenInformation, error) {
	rq, err := http.NewRequest("GET", f.BaseUrl+"/api/v1/tokens", nil)

	if err != nil {
		return nil, err
	}
	rq.Header.Add("Content-Type", "application/json")
	body, err := f.requestWithResponse(rq)
	if err != nil {
		return nil, err
	}

	response := model.ListTokenResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (f *FulcrumApiClient) RegenerateToken(tokenId string) (*model.TokenData, error) {
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/tokens/"+tokenId+"/regenerate", nil)
	if err != nil {
		return nil, err
	}
	body, err := f.requestWithResponse(rq)
	if err != nil {
		return nil, err
	}
	tokenData := model.TokenData{}
	err = json.Unmarshal(body, &tokenData)
	if err != nil {
		return nil, err
	}

	return &tokenData, nil
}

func (f *FulcrumApiClient) GetPendingJobs(agentToken string) ([]model.PendingJob, error) {
	rq, err := http.NewRequest("GET", f.BaseUrl+"/api/v1/jobs/pending", nil)
	if err != nil {
		return nil, err
	}
	body, err := f.requestWithResponseWithKey(rq, agentToken)
	if err != nil {
		return nil, err
	}

	var jobs []model.PendingJob
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, fmt.Errorf("Error parsing response body: %v\n", err)
	}

	return jobs, nil

}

func (f *FulcrumApiClient) ClaimJob(agentToken string, jobId string) error {
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/jobs/"+jobId+"/claim", nil)
	if err != nil {
		return err
	}

	_, err = f.requestWithResponseWithKey(rq, agentToken)
	if err != nil {
		return err
	}
	return nil
}

func (f *FulcrumApiClient) FinalizeJob(agentToken string, jobId string) error {
	body := `{
		"externalId": "go-provisioner-%s",
		"resources":{}
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/jobs/"+jobId+"/complete", strings.NewReader(fmt.Sprintf(body, uuid.New().String())))

	if err != nil {
		return err
	}
	_, err = f.requestWithResponseWithKey(rq, agentToken)
	if err != nil {
		return err
	}
	return nil
}

func (f *FulcrumApiClient) requestWithResponseWithKey(rq *http.Request, apiKey string) ([]byte, error) {
	rq.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := f.HttpClient.Do(rq)

	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Unexpected status code: %d\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v\n", err)
	}
	return body, nil
}

func (f *FulcrumApiClient) requestWithResponse(rq *http.Request) ([]byte, error) {
	return f.requestWithResponseWithKey(rq, f.ApiKey)
}

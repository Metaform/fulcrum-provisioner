package clients

import (
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
	GetPendingJobs() ([]model.PendingJob, error)
	ClaimJob(jobId string) error
	FinalizeJob(jobId string) error
}

type FulcrumApiClient struct {
	//agentToken string
	//httpClient *http.Client
	//baseUrl    string

	config.ApiConfig
}

func NewFulcrumApiClient(baseUrl string, token string) FulcrumApi {
	return &FulcrumApiClient{
		ApiConfig: config.ApiConfig{
			ApiKey:     token,
			HttpClient: http.Client{Timeout: 15 * time.Second},
			BaseUrl:    baseUrl,
		},
	}
}

func (f *FulcrumApiClient) GetPendingJobs() ([]model.PendingJob, error) {
	rq, err := http.NewRequest("GET", f.BaseUrl+"/api/v1/jobs/pending", nil)
	if err != nil {
		return nil, err
	}
	body, err := f.requestWithResponse(rq)
	if err != nil {
		return nil, err
	}

	var jobs []model.PendingJob
	if err := json.Unmarshal(body, &jobs); err != nil {
		return nil, fmt.Errorf("Error parsing response body: %v\n", err)
	}

	return jobs, nil

}

func (f *FulcrumApiClient) ClaimJob(jobId string) error {
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/jobs/"+jobId+"/claim", nil)
	if err != nil {
		return err
	}

	_, err = f.requestWithResponse(rq)
	if err != nil {
		return err
	}
	return nil
}

func (f *FulcrumApiClient) FinalizeJob(jobId string) error {
	body := `{
		"externalId": "go-provisioner-%s",
		"resources":{}
	}`
	rq, err := http.NewRequest("POST", f.BaseUrl+"/api/v1/jobs/"+jobId+"/complete", strings.NewReader(fmt.Sprintf(body, uuid.New().String())))

	if err != nil {
		return err
	}
	_, err = f.requestWithResponse(rq)
	if err != nil {
		return err
	}
	return nil
}

func (f *FulcrumApiClient) requestWithResponse(rq *http.Request) ([]byte, error) {

	rq.Header.Add("Authorization", "Bearer "+f.ApiKey)

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

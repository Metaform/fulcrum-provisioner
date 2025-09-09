package clients

import (
	"k8s-provisioner/clients/config"
)

type ManagementApi interface {
	CreateAsset(body string) (string, error)
	CreatePolicy(body string) (string, error)
	CreateContractDefinition(body string) (string, error)
	CreateSecret(body string) (string, error)
}

type ManagementApiClient struct {
	config.ApiConfig
}

func (i *ManagementApiClient) CreateAsset(body string) (string, error) {
	return config.SendRequest(i.HttpClient, i.ApiKey, body, i.BaseUrl+"/assets")
}
func (i *ManagementApiClient) CreatePolicy(body string) (string, error) {
	return config.SendRequest(i.HttpClient, i.ApiKey, body, i.BaseUrl+"/policydefinitions")
}

func (i *ManagementApiClient) CreateContractDefinition(body string) (string, error) {
	url := i.BaseUrl + "/contractdefinitions"
	return config.SendRequest(i.HttpClient, i.ApiKey, body, url)
}

func (i *ManagementApiClient) CreateSecret(body string) (string, error) {
	return config.SendRequest(i.HttpClient, i.ApiKey, body, i.BaseUrl+"/secrets")
}

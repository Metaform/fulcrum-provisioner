package clients

import (
	"k8s-provisioner/clients/config"
)

type IssuerApi interface {
	CreateHolder(did string, holderId string, name string) (string, error)
}

type IssuerApiClient struct {
	config.ApiConfig
}

func (i *IssuerApiClient) CreateHolder(did string, holderId string, name string) error {
	url := i.BaseUrl + "/holders"

	body := `{
				"did": "` + did + `",
    			"holderId": "` + holderId + `",
 				"name": "` + name + `"
			}`
	_, err := config.SendRequest(i.HttpClient, i.ApiKey, body, url)
	return err
}

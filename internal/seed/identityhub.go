package seed

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"k8s-provisioner/clients/config"
	identity "k8s-provisioner/clients/identity"
	mgmt "k8s-provisioner/clients/management"
	"k8s-provisioner/internal/model"
	"log"
	"strings"
)

//go:embed templates/participant.json
var participantJson string

func IdentityHubData(definition model.ParticipantDefinition) {
	kubernetesHost := definition.KubernetesIngressHost
	namespace := definition.ParticipantName

	identityApi := identity.IdentityApiClient{
		ApiConfig: config.ApiConfig{
			BaseUrl:    "http://" + kubernetesHost + "/" + namespace + "/cs/api/identity/v1alpha",
			ApiKey:     apiKey,
			HttpClient: config.CreateHttpClient(),
		},
	}
	ihBaseUrl := fmt.Sprintf("http://identityhub.%s.svc.cluster.local:7082", namespace)
	edcUrl := fmt.Sprintf("http://controlplane.%s.svc.cluster.local:8082", namespace)
	// Work on a local copy to avoid mutating global embedded template
	body := participantJson
	body = strings.Replace(body, "${PARTICIPANT_NAME}", definition.ParticipantName, -1)
	body = strings.Replace(body, "${PARTICIPANT_DID}", definition.Did, -1)
	body = strings.Replace(body, "${PARTICIPANT_DID_BASE64}", base64.StdEncoding.EncodeToString([]byte(definition.Did)), -1)
	body = strings.Replace(body, "${IH_BASE_URL}", ihBaseUrl, -1)
	body = strings.Replace(body, "${EDC_BASE_URL}", edcUrl, -1)

	participant, err := identityApi.CreateParticipant(body)
	if err != nil {
		log.Println(err)
		return
	}
	if participant == nil {
		log.Println("participant already exists")
		return
	}

	var mgmtApi = mgmt.ManagementApiClient{
		ApiConfig: config.ApiConfig{
			HttpClient: config.CreateHttpClient(),
			BaseUrl:    "http://" + kubernetesHost + "/" + namespace + "/cp/api/management/v3",
			ApiKey:     "password",
		},
	}
	secretBody := `
	{
		"@context": [
			"https://w3id.org/edc/connector/management/v0.0.1"
		],
		"@id": "${ID}",
		"value": "${SECRET}"
    }`
	secretBody = strings.Replace(secretBody, "${ID}", participant.ClientId+"-sts-client-secret", -1)
	secretBody = strings.Replace(secretBody, "${SECRET}", participant.ClientSecret, -1)

	_, err = mgmtApi.CreateSecret(secretBody)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("participant created")
}

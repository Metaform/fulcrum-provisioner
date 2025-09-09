package seed

import (
	"encoding/base64"
	"fmt"
	"k8s-provisioner/clients/config"
	"k8s-provisioner/clients/issuer"
	"k8s-provisioner/internal/model"
	"net/http"
	"time"
)

func IssuerData(definition model.ParticipantDefinition) {
	kubernetesHost := definition.KubernetesIngressHost
	issuerId := "did:web:dataspace-issuer-service.poc-issuer.svc.cluster.local%3A10016:issuer"
	issuerB64 := base64.StdEncoding.EncodeToString([]byte(issuerId))
	issuerApi := clients.IssuerApiClient{
		ApiConfig: config.ApiConfig{
			BaseUrl:    "http://" + kubernetesHost + "/issuer/ad/api/admin/v1alpha/participants/" + issuerB64,
			ApiKey:     "c3VwZXItdXNlcg==.c3VwZXItc2VjcmV0LWtleQo=",
			HttpClient: http.Client{Timeout: 15 * time.Second},
		},
	}

	err := issuerApi.CreateHolder(definition.Did, definition.Did, definition.ParticipantName)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("issuer account created for participant ", definition.ParticipantName)
}

package seed

import (
	_ "embed"
	"k8s-provisioner/clients/config"
	clients "k8s-provisioner/clients/management"
	"k8s-provisioner/internal/model"
	"log"
)

// todo: make configurable
const apiKey = "c3VwZXItdXNlcg==.c3VwZXItc2VjcmV0LWtleQo="

//go:embed resources/asset1.json
var asset1Json string

//go:embed resources/asset2.json
var asset2json string

//go:embed resources/policy_dataprocessor.json
var policyDataProcessorJson string

//go:embed resources/policy_membership.json
var policyMembershipJson string

//go:embed resources/policy_sensitive_data.json
var policySensitiveDataJson string

//go:embed resources/contractdef_require_membership.json
var defRequireMembership string

//go:embed resources/contractdef_require_sensitive.json
var defSensitive string

func ConnectorData(definition model.ParticipantDefinition) {

	kubernetesHost := definition.KubernetesIngressHost
	namespace := definition.ParticipantName

	mgmtApi := clients.ManagementApiClient{
		ApiConfig: config.ApiConfig{
			BaseUrl:    "http://" + kubernetesHost + "/" + namespace + "/cp/api/management/v3",
			ApiKey:     "password",
			HttpClient: config.CreateHttpClient(),
		},
	}

	// create assets
	for _, asset := range []string{asset1Json, asset2json} {
		_, err := mgmtApi.CreateAsset(asset)
		if err != nil {
			log.Println(err)
			return
		}

	}
	log.Println("assets created")

	// create policies
	for _, policy := range []string{policyDataProcessorJson, policyMembershipJson, policySensitiveDataJson} {
		_, err := mgmtApi.CreatePolicy(policy)
		if err != nil {
			log.Println(err)
			return
		}
	}
	log.Println("policies created")

	// create contract defs
	for _, cd := range []string{defRequireMembership, defSensitive} {
		_, err := mgmtApi.CreateContractDefinition(cd)
		if err != nil {
			log.Println(err)
			return
		}
	}
	log.Println("contract definitions created")

}

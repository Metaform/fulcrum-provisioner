package provisioner

import (
	"context"
	_ "embed"
	"k8s-provisioner/internal/kube"
	"k8s-provisioner/internal/model"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// ProvisioningAgent manages resources on a Kubernetes cluster
type ProvisioningAgent interface {
	CreateResources(model.ParticipantDefinition, func(definition model.ParticipantDefinition)) (map[string]string, error)
	DeleteResources(model.ParticipantDefinition) (map[string]string, error)
}

// Centralize deployment names used for readiness checks
var participantDeploymentNames = []string{"controlplane", "identityhub", "dataplane"}

type ProvisioningAgentImpl struct {
	ctx        context.Context
	kubeClient client.Client
}

//go:embed templates/connector.yaml
var participantYaml string

//go:embed templates/identityhub.yaml
var identityhubYaml string

func NewProvisioningAgent(context context.Context, kubeClient client.Client) ProvisioningAgent {
	return &ProvisioningAgentImpl{
		ctx:        context,
		kubeClient: kubeClient,
	}
}

func (p ProvisioningAgentImpl) CreateResources(definition model.ParticipantDefinition, readyCallback func(model.ParticipantDefinition)) (map[string]string, error) {
	resources1, e1 := p.applyYaml(&definition.ParticipantName, &definition.Did, participantYaml, p.applyResource)
	if e1 != nil {
		return nil, e1
	}
	resources2, e2 := p.applyYaml(&definition.ParticipantName, &definition.Did, identityhubYaml, p.applyResource)
	if e2 != nil {
		return nil, e2
	}
	// Merge maps
	mergedResources := make(map[string]string)
	for k, v := range resources1 {
		mergedResources[k] = v
	}
	for k, v := range resources2 {
		mergedResources[k] = v
	}

	// Introduce a clear variable for namespace usage
	namespace := definition.ParticipantName

	// Start readiness wait in a separate goroutine (non-blocking definition)
	kube.WaitForDeploymentsAsync(
		p.kubeClient,
		p.ctx,
		namespace,
		participantDeploymentNames,
		func() {
			readyCallback(definition)
		},
	)
	return mergedResources, nil
}

func (p ProvisioningAgentImpl) DeleteResources(definition model.ParticipantDefinition) (map[string]string, error) {
	resources1, e1 := p.applyYaml(&definition.ParticipantName, &definition.Did, participantYaml, p.deleteResource)
	if e1 != nil {
		return nil, e1
	}
	resources2, e2 := p.applyYaml(&definition.ParticipantName, &definition.Did, identityhubYaml, p.deleteResource)
	if e2 != nil {
		return nil, e2
	}
	// Merge maps
	mergedResources := make(map[string]string)
	for k, v := range resources1 {
		mergedResources[k] = v
	}
	for k, v := range resources2 {
		mergedResources[k] = v
	}
	return mergedResources, nil
}

func (p ProvisioningAgentImpl) applyYaml(participantName *string, did *string, yamlString string, kubernetesAction action) (map[string]string, error) {
	yamlString = strings.Replace(yamlString, "${PARTICIPANT_NAME}", *participantName, -1)
	yamlString = strings.Replace(yamlString, "$PARTICIPANT_NAME", *participantName, -1)
	yamlString = strings.Replace(yamlString, "${PARTICIPANT_ID}", *did, -1)
	yamlString = strings.Replace(yamlString, "$PARTICIPANT_ID", *did, -1)

	docs := strings.Split(yamlString, "---")

	resourceMap := make(map[string]string)
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			return nil, err
		}

		resourceMap[obj.GetName()] = obj.GetKind()
		err := kubernetesAction(p.kubeClient, p.ctx, obj)
		if err != nil {
			return nil, err
		}
	}
	return resourceMap, nil
}

func (p ProvisioningAgentImpl) applyResource(c client.Client, ctx context.Context, object client.Object) error {
	// Server-Side Apply
	err := c.Patch(
		ctx,
		object,
		client.Apply,
		client.FieldOwner("go-provisioner"),
		// Optional: take ownership of fields (overwrites conflicts)
		client.ForceOwnership,
	)
	return err
}

type action func(client.Client, context.Context, client.Object) error

func (p ProvisioningAgentImpl) deleteResource(c client.Client, ctx context.Context, object client.Object) error {
	return c.Delete(ctx, object)
}

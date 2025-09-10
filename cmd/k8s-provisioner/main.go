// Go
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"k8s-provisioner/clients/fulcrum"
	"k8s-provisioner/internal/model"
	"k8s-provisioner/internal/provisioner"
	"k8s-provisioner/internal/seed"
	"k8s-provisioner/internal/server"
	"k8s-provisioner/internal/store"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gofiber/fiber/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	_ "embed"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var agentStore store.AgentInfoStore

type CLI struct {
	KubeConfig  string `help:"Path to KubeConfig file" env:"KUBECONFIG" default:"~/.kube/config"`
	FulcrumCore string `help:"Fulcrum Core API Host" env:"FULCRUM_CORE"`
	Postgres    string `help:"Postgres connection string" env:"PG_CONNECTION_STRING"`
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	// Create context with cancellation
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	kubeClient, err := initializeKubeClient(cli.KubeConfig)
	if err != nil {
		log.Fatalf("create kube client: %v", err)
	}
	provisioningAgent := provisioner.NewProvisioningAgent(ctx, kubeClient)

	db, err := sql.Open("pgx", cli.Postgres)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	// Ensure DB schema exists before using the store
	if err := store.EnsureSchema(ctx, db); err != nil {
		log.Fatalf("ensure db schema: %v", err)
	}

	agentStore = store.NewPostgresAgentInfoStore(db)
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	// Start a periodic health check
	if cli.FulcrumCore == "" {
		log.Printf("No Fulcrum Core API endpoint was supplied, will skip periodic checking")
	} else {
		apiClient := clients.NewFulcrumApiClient(cli.FulcrumCore)
		token, seedError := seedFulcrumCore(apiClient)
		if seedError != nil {
			log.Fatalf("Error seeding Fulcrum Core: %s", seedError)
		}
		if token == nil {
			log.Fatalf("Error seeding/fetching fulcrum token: token is nil")
		}

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pollFulcrum(apiClient, *token, provisioningAgent)
				}
			}
		}()
		log.Println("Start polling Fulcrum Core at " + cli.FulcrumCore)
	}

	app := fiber.New()
	{
		group := app.Group("/api/v1/resources")
		group.Post("/", server.CreateResource(provisioningAgent, onDeploymentReady))
		group.Delete("/", server.DeleteResource(provisioningAgent))
	}
	// Run server and shut down gracefully on ctx cancel
	go func() {
		if err := app.Listen(":9999"); err != nil {
			log.Printf("fiber server error: %v", err)
		}
	}()
	<-ctx.Done()
	log.Println("\nGracefully shutting down...")
	_ = app.Shutdown()
}

func initializeKubeClient(kubeConfigPath string) (client.Client, error) {
	exists := true
	konfig := &rest.Config{}

	if kubeConfigPath == "" {
		exists = false
	} else if _, err := os.Stat(kubeConfigPath); errors.Is(err, os.ErrNotExist) {
		log.Printf("kubeconfig file %s does not exist, falling back to in-cluster config\n", kubeConfigPath)
		exists = false
	}
	if exists {

		// Load kubeconfig (or use in-cluster if applicable)
		log.Println("Load kubeconfig from ", kubeConfigPath, "")
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			log.Fatalf("load kubeconfig: %v", err)
		}
		konfig = cfg
	} else {
		log.Println("No kubeconfig provided, using in-cluster config", "")
		cfg, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalf("load in-cluster config: %v", err)
		}
		konfig = cfg
	}

	// Scheme with core types
	// --- Prepare scheme ---
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	kubeClient, err := client.New(konfig, client.Options{Scheme: scheme})
	return kubeClient, err
}

func pollFulcrum(apiClient clients.FulcrumApi, agentToken string, agent provisioner.ProvisioningAgent) {

	jobs, err := apiClient.GetPendingJobs(agentToken)
	if err != nil {
		log.Printf("Error getting pending jobs: %s\n", err)
		return
	}
	if len(jobs) > 0 {
		log.Println("Got " + strconv.Itoa(len(jobs)) + " pending jobs")
	}

	for _, job := range jobs {
		if job.Status == "Pending" {
			def := model.ParticipantDefinition{
				ParticipantName:       fmt.Sprintf("%v", job.Service.Properties["participantName"]),
				Did:                   fmt.Sprintf("%v", job.Service.Properties["participantDid"]),
				KubernetesIngressHost: fmt.Sprintf("%v", job.Service.Properties["kubeHost"]),
			}
			e := apiClient.ClaimJob(agentToken, job.Id)
			log.Printf("Claimed job %s (\"%s\"), Action = %s\n", job.Id, job.Service.Name, job.Action)
			if e != nil {
				log.Printf("Error claiming job: %s", e)
			}

			if job.Action == "Create" {
				_, provisioningError := agent.CreateResources(def, func(definition model.ParticipantDefinition) {
					onDeploymentReady(definition)
					e = apiClient.FinalizeJob(agentToken, job.Id)
					if e != nil {
						log.Printf("Error finalizing job: %s\n", e)
					} else {
						log.Printf("Finalized job: %s\n", job.Id)
					}
				})
				if provisioningError != nil {
					log.Printf("Error creating resources: %s\n", provisioningError)
					return
				}
			} else if job.Action == "Delete" {
				_, provisioningErr := agent.DeleteResources(def)
				if provisioningErr != nil {
					log.Printf("Error creating resources: %s\n", provisioningErr)
					return
				} else {
					log.Println("Resource deletion complete.")
					e = apiClient.FinalizeJob(agentToken, job.Id)
					if e != nil {
						log.Printf("Error finalizing job: %s\n", e)
					} else {
						log.Printf("Finalized job: %s\n", job.Id)
					}
				}
			}

		} else {
			log.Printf("Pending Job in status %s", job.Status)
		}
	}
}

func onDeploymentReady(definition model.ParticipantDefinition) {
	log.Println("Deployments ready in namespace", definition.ParticipantName, "-> creating data")

	seed.ConnectorData(definition)
	seed.IdentityHubData(definition)
	seed.IssuerData(definition)

	log.Println("Data seeding complete in namespace", definition.ParticipantName)

}

func seedFulcrumCore(apiClient clients.FulcrumApi) (*string, error) {
	const agentName = "EDC Provisioner Agent"
	const tokenName = "Provisioner Access Token"

	ctx := context.Background()
	ai, err := agentStore.GetByName(ctx, agentName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			//seed basic fulcrum data
			err = createAgent(agentName, apiClient, &ai)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// create agent token
	log.Println("  > creating agent token")
	token, e := apiClient.CreateAgentToken(ai.AgentId, tokenName)
	if e != nil {
		return nil, fmt.Errorf("failed to create agent token: %w", e)
	}
	ai.TokenId = token.Id
	return &token.Value, agentStore.Upsert(ctx, ai)

}

func createAgent(name string, apiClient clients.FulcrumApi, receptacle *store.AgentInfo) error {
	log.Println("### Seeding Fulcrum Core ###")
	// seed service type
	log.Println("  > creating service type")

	serviceTypeId, err := apiClient.CreateServiceType("edc-aio", "EDC All-in-one deployment")
	if err != nil {
		return fmt.Errorf("failed to create service type: %w", err)
	}

	//create agent-type
	log.Println("  > creating agent type")
	agentTypeId, err := apiClient.CreateAgentType(serviceTypeId, "go-provisioner-agent")
	if err != nil {
		return fmt.Errorf("failed to create agent type: %w", err)
	}

	// create participant
	log.Println("  > creating participant")
	participantId, err := apiClient.CreateParticipant("K8S Provisioner Participant")
	if err != nil {
		return fmt.Errorf("failed to create participant: %w", err)
	}

	// create service-group
	log.Println("  > creating service group")
	serviceGroupId, err := apiClient.CreateServiceGroup(participantId, "EDC Services Group")
	if err != nil {
		return fmt.Errorf("failed to create service group: %w", err)
	}
	log.Println("Created service group", serviceGroupId)

	// create agent
	log.Println("  > creating agent")
	agentId, err := apiClient.CreateAgent(model.AgentData{
		Name:          name,
		ProviderId:    participantId,
		AgentTypeId:   agentTypeId,
		Tags:          []string{"cfm", "edc"},
		Configuration: make(map[string]interface{}),
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	receptacle.AgentId = agentId
	receptacle.ProviderId = participantId
	receptacle.AgentTypeId = agentTypeId
	receptacle.Name = name
	receptacle.ServiceTypeId = serviceTypeId
	receptacle.ServiceGroupId = serviceGroupId
	return nil
}

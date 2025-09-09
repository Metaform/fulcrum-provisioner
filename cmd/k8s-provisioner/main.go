// Go
package main

import (
	"context"
	"errors"
	"fmt"
	"k8s-provisioner/clients/fulcrum"
	"k8s-provisioner/internal/model"
	"k8s-provisioner/internal/provisioner"
	"k8s-provisioner/internal/seed"
	"k8s-provisioner/internal/server"
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
)

type CLI struct {
	KubeConfig  string `help:"Path to KubeConfig file" env:"KUBECONFIG" default:"~/.kube/config"`
	Token       string `help:"Fulcrum Core API agent Token" env:"TOKEN"`
	FulcrumCore string `help:"Fulcrum Core API Host" env:"FULCRUM"`
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	// Create context with cancellation
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	konfig := &rest.Config{}
	exists := true
	if cli.KubeConfig == "" {
		exists = false
	} else if _, err := os.Stat(cli.KubeConfig); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("kubeconfig file %s does not exist, falling back to in-cluster config\n", cli.KubeConfig)
		exists = false
	}
	if exists {

		// Load kubeconfig (or use in-cluster if applicable)
		fmt.Println("Load kubeconfig from ", cli.KubeConfig, "")
		cfg, err := clientcmd.BuildConfigFromFlags("", cli.KubeConfig)
		if err != nil {
			log.Fatalf("load kubeconfig: %v", err)
		}
		konfig = cfg
	} else {
		fmt.Println("No kubeconfig provided, using in-cluster config", "")
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
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	provisioningAgent := provisioner.NewProvisioningAgent(ctx, kubeClient)

	// Start periodic health check
	if cli.Token != "" && cli.FulcrumCore != "" {
		apiClient := clients.NewFulcrumApiClient(cli.FulcrumCore, cli.Token)

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					pollFulcrum(apiClient, provisioningAgent)
				}
			}
		}()
		fmt.Println("Start polling Fulcrum Core at " + cli.FulcrumCore)
	} else {
		fmt.Printf("No Fulcrum Agent token and/or host address was supplied, will skip periodic checking")
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
	fmt.Println("\nGracefully shutting down...")
	_ = app.Shutdown()
}

func pollFulcrum(apiClient clients.FulcrumApi, agent provisioner.ProvisioningAgent) {

	jobs, err := apiClient.GetPendingJobs()
	if err != nil {
		fmt.Printf("Error getting pending jobs: %s\n", err)
		return
	}
	if len(jobs) > 0 {
		fmt.Println("Got " + strconv.Itoa(len(jobs)) + " pending jobs")
	}

	for _, job := range jobs {
		if job.Status == "Pending" {
			def := model.ParticipantDefinition{
				ParticipantName:       fmt.Sprintf("%v", job.Service.Properties["participantName"]),
				Did:                   fmt.Sprintf("%v", job.Service.Properties["participantDid"]),
				KubernetesIngressHost: fmt.Sprintf("%v", job.Service.Properties["kubeHost"]),
			}
			e := apiClient.ClaimJob(job.Id)
			fmt.Printf("Claimed job %s (\"%s\"), Action = %s\n", job.Id, job.Service.Name, job.Action)
			if e != nil {
				fmt.Printf("Error claiming job: %s", e)
			}

			if job.Action == "Create" {
				_, provisioningError := agent.CreateResources(def, func(definition model.ParticipantDefinition) {
					e = apiClient.FinalizeJob(job.Id)
					if e != nil {
						fmt.Printf("Error finalizing job: %s\n", e)
					} else {
						fmt.Printf("Finalized job: %s\n", job.Id)
					}
				})
				if provisioningError != nil {
					fmt.Printf("Error creating resources: %s\n", provisioningError)
					return
				}
			} else if job.Action == "Delete" {
				_, provisioningErr := agent.DeleteResources(def)
				if provisioningErr != nil {
					fmt.Printf("Error creating resources: %s\n", provisioningErr)
					return
				} else {
					fmt.Println("Resource deletion complete.")
					e = apiClient.FinalizeJob(job.Id)
					if e != nil {
						fmt.Printf("Error finalizing job: %s\n", e)
					} else {
						fmt.Printf("Finalized job: %s\n", job.Id)
					}
				}
			}

		} else {
			fmt.Printf("Pending Job in status %s", job.Status)
		}
	}
}

func onDeploymentReady(definition model.ParticipantDefinition) {
	fmt.Println("Deployments ready in namespace", definition.ParticipantName, "-> seeding data")

	seed.ConnectorData(definition)
	seed.IdentityHubData(definition)
	seed.IssuerData(definition)

	fmt.Println("Data seeding complete in namespace", definition.ParticipantName)

}

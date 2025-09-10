package server

import (
	"k8s-provisioner/internal/model"
	"k8s-provisioner/internal/provisioner"
	"log"

	"github.com/gofiber/fiber/v2"
)

func CreateResource(provisioningAgent provisioner.ProvisioningAgent, onDeploymentReady func(definition model.ParticipantDefinition)) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		definition := model.ParticipantDefinition{
			KubernetesIngressHost: "localhost",
		}
		if err := c.BodyParser(&definition); err != nil {
			return err
		}

		log.Println("Creating resources")
		mergedResources, err2 := provisioningAgent.CreateResources(definition, onDeploymentReady)
		if err2 != nil {
			return err2
		}

		return c.JSON(mergedResources)

	}
}

func DeleteResource(provisioningAgent provisioner.ProvisioningAgent) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var request model.ParticipantDefinition
		if err := c.BodyParser(&request); err != nil {
			return err
		}
		log.Println("Deleting resources")
		mergedResources, err2 := provisioningAgent.DeleteResources(request)
		if err2 != nil {
			return err2
		}

		return c.JSON(mergedResources)
	}
}

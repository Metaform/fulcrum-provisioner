package kube

import (
	"context"
	"log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// todo: make configurable
const readinessPollInterval = 2 * time.Second

// WaitForDeploymentsAsync runs the readiness check in the background and invokes the callback on success.
func WaitForDeploymentsAsync(
	c client.Client,
	ctx context.Context,
	namespace string,
	deployments []string,
	callback func(),
) {
	log.Println("Waiting for deployments", deployments, "")
	go func() {
		if err := waitForDeployments(c, ctx, namespace, deployments); err != nil {
			log.Printf("deployment readiness check failed for namespace %s: %v\n", namespace, err)
			return
		}
		callback()
	}()
}

// waitForDeployments waits for all given deployments concurrently and returns an error if any fail.
func waitForDeployments(c client.Client, ctx context.Context, namespace string, deployments []string) error {
	errCh := make(chan error, len(deployments))
	for _, name := range deployments {
		name := name // capture
		go func() {
			errCh <- waitForDeployment(c, ctx, namespace, name)
		}()
	}
	var firstErr error
	for _, deployment := range deployments {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
		} else if err == nil {
			log.Println("Deployment", deployment, "ready")
		}
	}
	return firstErr
}

// waitForDeployment polls until the deployment reaches the desired ready replicas.
func waitForDeployment(c client.Client, ctx context.Context, namespace string, name string) error {
	deployment := &appsv1.Deployment{}
	for {
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
			return err
		}

		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}
		if deployment.Status.ReadyReplicas == desired {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(readinessPollInterval):
			continue
		}
	}
}

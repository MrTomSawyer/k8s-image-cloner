package config

import (
	"context"
	"flag"
	"fmt"

	appsv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config holds the configuration needed for the controller to operate.
// It includes Docker credentials and parameters for the controller behavior,
// as well as an API reader for accessing Kubernetes resources.
type Config struct {
	apiReader client.Reader
	DockerConfig
	ControllerParams
}

// DockerConfig stores Docker registry credentials.
type DockerConfig struct {
	Username string
	Password string
}

// ControllerParams holds the namespace and secret name used by the controller.
type ControllerParams struct {
	Namespace        string
	DockerSecretName string
}

// New creates a new Config instance.
func New(apiReader client.Reader) *Config {
	return &Config{apiReader: apiReader}
}

// ParseFlags parses command-line flags.
func (c *Config) ParseFlags() *Config {
	flag.StringVar(&c.ControllerParams.DockerSecretName, "docker-creds-secret", "image-cloner-creds", "Creds of a docker backup registry")
	flag.StringVar(&c.ControllerParams.Namespace, "namespace", "default", "Namespace where a controller will work")
	flag.Parse()

	return c
}

// RetrieveDockerSecrets fetches the Docker credentials stored in a Kubernetes secret.
func (c *Config) RetrieveDockerSecrets(ctx context.Context, nameSpace, secretName string) error {
	secret := &appsv1.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: nameSpace,
		Name:      secretName,
	}

	err := c.apiReader.Get(ctx, secretNamespaceName, secret)
	if err != nil {
		return fmt.Errorf("failed to fetch docker secret: %s", err.Error())
	}

	c.DockerConfig.Username = string(secret.Data["username"])
	c.DockerConfig.Password = string(secret.Data["password"])

	return nil
}

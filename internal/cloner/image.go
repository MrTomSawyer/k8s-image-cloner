package cloner

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageCloner is responsible for cloning container images from one registry to another.
type ImageCloner struct {
	client.Client
	apiReader client.Reader
	logger    logr.Logger
	сfg       authn.AuthConfig
}

// NewImageCloner creates a new ImageCloner instance
func NewImageCloner(client client.Client, apiReader client.Reader, logger logr.Logger) *ImageCloner {
	return &ImageCloner{
		Client:    client,
		apiReader: apiReader,
		logger:    logger,
	}
}

// SetupForDocker configures the Docker authentication credentials.
func (ic *ImageCloner) SetupForDocker(username, password string) error {
	ic.сfg = authn.AuthConfig{
		Username: username,
		Password: password,
	}

	return nil
}

// IsCloned checks if the image has already been cloned by inspecting the image name.
func (ic *ImageCloner) IsCloned(image string) bool {
	return strings.HasPrefix(image, ic.сfg.Username)
}

// Clone performs image cloning process.
func (ic *ImageCloner) Clone(ctx context.Context, originalImage, clonedImage string) error {
	ic.logger.Info("starting cloning process:", "image", originalImage)
	err := crane.Copy(originalImage, clonedImage, crane.WithAuth(authn.FromConfig(ic.сfg)))
	if err != nil {
		ic.logger.Error(err, "failed to clone the image")
		return err
	}

	return nil
}

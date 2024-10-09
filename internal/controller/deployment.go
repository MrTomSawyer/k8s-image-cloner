package controller

import (
	"context"
	"strings"

	"github.com/MrTomSawyer/k8s-image-controller/internal/controller/finalizers"
	"github.com/MrTomSawyer/k8s-image-controller/internal/controller/predicates"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const delProtectionFinalizerName = "image-cloner/del-protection"

// ImageCloner defines an interface for cloning images and checking if they are cloned.
type ImageCloner interface {
	Clone(ctx context.Context, originalImage, clonedImage string) error
	IsCloned(image string) bool
}

// DeploymentReconciler is responsible for reconciling DaemonSet resources in Kubernetes.
// It clones container images from the DaemonSet's containers to a backup registry if they are not already cloned.
type DeploymentReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	cloner       ImageCloner
	logger       logr.Logger
	clonedPrefix string
}

// NewDeploymentReconciler creates a new instance of DeploymentReconciler.
func NewDeploymentReconciler(client client.Client, scheme *runtime.Scheme, cloner ImageCloner, logger logr.Logger, clonedPrefix string) *DeploymentReconciler {
	return &DeploymentReconciler{
		Client:       client,
		Scheme:       scheme,
		cloner:       cloner,
		logger:       logger,
		clonedPrefix: clonedPrefix,
	}
}

// Register registers the DeploymentReconciler with the controller manager.
// It sets up event filters to ignore resources in the kube-system namespace, already cloned images, and unchanged Deploymants.
func (r *DeploymentReconciler) Register(mrg ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mrg).
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.NewPredicateFuncs(predicates.IgnoreKubeSystem)).
		WithEventFilter(predicate.NewPredicateFuncs(predicates.IgnoreDeploymentClonedImages(r.clonedPrefix))).
		WithEventFilter(predicates.IgnoreUnchangedDeploymentSpec()).
		Complete(r)
}

// Reconcile is the main reconciliation loop for Deployments. It checks if the container images are cloned,
// clones the images if needed, and updates the Deployment with the new image references.
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.WithValues("namespace", req.Namespace)

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, req.NamespacedName, deployment)
	if err != nil {
		logger.Error(err, "failed to get a deployment\n")
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		logger.Info("starting reconciling", "image", container.Image)

		if r.cloner.IsCloned(container.Image) {
			logger.Info("this image has already been cloned")
			continue
		}

		clonedImg := r.clonedPrefix + "/" + strings.ReplaceAll(container.Image, "/", "-")
		err := r.cloner.Clone(ctx, container.Image, clonedImg)
		if err != nil {
			logger.Error(err, "failed to clone the image\n")
			return ctrl.Result{}, err
		}

		deployment.Spec.Template.Spec.Containers[i].Image = clonedImg

		finalizers := finalizers.RemoveFinalizer(deployment.ObjectMeta.Finalizers, delProtectionFinalizerName)
		deployment.ObjectMeta.Finalizers = finalizers
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &appsv1.Deployment{}
		if err := r.Get(ctx, req.NamespacedName, latest); err != nil {
			logger.Error(err, "failed to get a deployment on retry\n")
			return err
		}

		latest.Spec.Template.Spec.Containers = deployment.Spec.Template.Spec.Containers
		latest.ObjectMeta.Finalizers = deployment.ObjectMeta.Finalizers
		return r.Update(ctx, latest)
	})
	if err != nil {
		logger.Error(err, "failed to update a deployment\n")
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation successfully finished")
	return ctrl.Result{}, nil
}

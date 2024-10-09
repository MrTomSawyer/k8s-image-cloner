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

// DaemonsetReconciler is responsible for reconciling DaemonSet resources in Kubernetes.
// It clones container images from the DaemonSet's containers to a backup registry if they are not already cloned.
type DaemonsetReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	cloner       ImageCloner
	logger       logr.Logger
	clonedPrefix string
}

// NewDaemonsetReconciler creates a new instance of DaemonsetReconciler.
func NewDaemonsetReconciler(client client.Client, scheme *runtime.Scheme, cloner ImageCloner, logger logr.Logger, clonedPrefix string) *DaemonsetReconciler {
	return &DaemonsetReconciler{
		Client:       client,
		Scheme:       scheme,
		cloner:       cloner,
		logger:       logger,
		clonedPrefix: clonedPrefix,
	}
}

// Register registers the DaemonsetReconciler with the controller manager.
// It sets up event filters to ignore resources in the kube-system namespace, already cloned images, and unchanged DaemonSets.
func (r *DaemonsetReconciler) Register(mrg ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mrg).
		For(&appsv1.DaemonSet{}).
		WithEventFilter(predicate.NewPredicateFuncs(predicates.IgnoreKubeSystem)).
		WithEventFilter(predicate.NewPredicateFuncs(predicates.IgnoreDaemonsetClonedImages(r.clonedPrefix))).
		WithEventFilter(predicates.IgnoreUnchangedDaemonsetSpec()).
		Complete(r)
}

// Reconcile is the main reconciliation loop for DaemonSets. It checks if the container images are cloned,
// clones the images if needed, and updates the DaemonSet with the new image references.
func (r *DaemonsetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.WithValues("namespace", req.Namespace)

	daemonset := &appsv1.DaemonSet{}
	err := r.Get(ctx, req.NamespacedName, daemonset)
	if err != nil {
		logger.Error(err, "failed to get a daemonset")
		return ctrl.Result{}, nil
	}

	for i, container := range daemonset.Spec.Template.Spec.Containers {
		logger.Info("starting reconciling", "image", container.Image)

		if r.cloner.IsCloned(container.Image) {
			logger.Info("this image has already been cloned")
			continue
		}

		clonedImg := r.clonedPrefix + "/" + strings.ReplaceAll(container.Image, "/", "-")
		err := r.cloner.Clone(ctx, container.Image, clonedImg)
		if err != nil {
			logger.Error(err, "failed to clone the image")
			return ctrl.Result{}, err
		}

		daemonset.Spec.Template.Spec.Containers[i].Image = clonedImg

		finalizers := finalizers.RemoveFinalizer(daemonset.ObjectMeta.Finalizers, delProtectionFinalizerName)
		daemonset.ObjectMeta.Finalizers = finalizers
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &appsv1.DaemonSet{}
		if err := r.Get(ctx, req.NamespacedName, latest); err != nil {
			logger.Error(err, "failed to get a daemonset on retry\n")
			return err
		}

		latest.Spec.Template.Spec.Containers = daemonset.Spec.Template.Spec.Containers
		latest.ObjectMeta.Finalizers = daemonset.ObjectMeta.Finalizers
		return r.Update(ctx, latest)
	})
	if err != nil {
		logger.Error(err, "failed to update a daemonset")
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation successfully finished")
	return ctrl.Result{}, nil
}

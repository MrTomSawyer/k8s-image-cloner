package predicates

import (
	"log"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// IgnoreKubeSystem is a helper function that returns true if the object is NOT in the "kube-system" namespace.
// It is used to filter out resources from the "kube-system" namespace during reconciliation.
func IgnoreKubeSystem(object client.Object) bool {
	return object.GetNamespace() != "kube-system"
}

// IgnoreDeploymentClonedImages returns a function that checks if the container image of a Deployment does NOT
// have the specified cloned image prefix. It is used to filter out Deployments that have already been cloned.
func IgnoreDeploymentClonedImages(imgPrefix string) func(object client.Object) bool {
	return func(object client.Object) bool {
		deployment, ok := object.(*appsv1.Deployment)
		if !ok {
			log.Printf("object is not a deployment")
			return false
		}

		img := deployment.Spec.Template.Spec.Containers[0].Image
		return !strings.HasPrefix(img, imgPrefix)
	}
}

// IgnoreDaemonsetClonedImages returns a function that checks if the container image of a DaemonSet does NOT
// have the specified cloned image prefix. It is used to filter out DaemonSets that have already been cloned.
func IgnoreDaemonsetClonedImages(imgPrefix string) func(object client.Object) bool {
	return func(object client.Object) bool {
		deployment, ok := object.(*appsv1.DaemonSet)
		if !ok {
			log.Printf("object is not a daemonset")
			return false
		}

		img := deployment.Spec.Template.Spec.Containers[0].Image
		return !strings.HasPrefix(img, imgPrefix)
	}
}

// IgnoreUnchangedDeploymentSpec returns a predicate.Predicate that filters out update events for Deployments
// where the spec has not changed. This is used to avoid reconciling Deployments unnecessarily.
func IgnoreUnchangedDeploymentSpec() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSpec := e.ObjectOld.(*appsv1.Deployment).Spec
			newSpec := e.ObjectNew.(*appsv1.Deployment).Spec
			return !reflect.DeepEqual(oldSpec, newSpec)
		},
	}
}

// IgnoreUnchangedDaemonsetSpec returns a predicate.Predicate that filters out update events for DaemonSets
// where the spec has not changed. This is used to avoid reconciling DaemonSets unnecessarily.
func IgnoreUnchangedDaemonsetSpec() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSpec := e.ObjectOld.(*appsv1.DaemonSet).Spec
			newSpec := e.ObjectNew.(*appsv1.DaemonSet).Spec
			return !reflect.DeepEqual(oldSpec, newSpec)
		},
	}
}

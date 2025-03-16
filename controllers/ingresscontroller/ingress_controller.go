/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingresscontroller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kuoss/ingress-annotator/pkg/matcher"
	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/util"
)

type ingressScope struct {
	logger    logr.Logger
	namespace *corev1.Namespace
	ingress   *networkingv1.Ingress
}

type IngressReconciler struct {
	client.Client
	*matcher.Matcher
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	// Fetch Ingress resource
	var ingress networkingv1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deleted ingresses
	if !ingress.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Fetch Namespace resource
	var namespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: ingress.Namespace}, &namespace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize ingressScope
	scope := &ingressScope{
		logger:    logger,
		namespace: &namespace,
		ingress:   &ingress,
	}

	// Reconcile Ingress
	return r.reconcileIngress(ctx, scope)
}

func (r *IngressReconciler) reconcileIngress(ctx context.Context, scope *ingressScope) (ctrl.Result, error) {
	ingress := scope.ingress
	logger := scope.logger

	toBeAnnotations := r.GetToBeAnnotations(ctx, scope)

	// Early exit silently if there are no changes to annotations.
	if annotationsEqual(toBeAnnotations, ingress.Annotations) {
		return ctrl.Result{}, nil
	}

	ingress.Annotations = toBeAnnotations
	if err := r.Update(ctx, ingress); err != nil {
		logger.Error(err, "Failed to update Ingress with to-be annotations")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	logger.Info("Successfully reconciled Ingress with to-be annotations")
	return ctrl.Result{}, nil
}

func (r *IngressReconciler) GetToBeAnnotations(ctx context.Context, scope *ingressScope) model.Annotations {
	ingress := scope.ingress
	logger := scope.logger

	toBeAnnotations := copyAnnotations(ingress.Annotations)
	delete(toBeAnnotations, model.ReconcileKey)

	// remove managed annotations if exists
	if value, exists := toBeAnnotations[model.ManagedAnnotationsKey]; exists {
		managedAnnotations := make(model.Annotations)
		if err := json.Unmarshal([]byte(value), &managedAnnotations); err != nil {
			logger.Error(err, "Warning: Failed to unmarshal managed annotations")
		} else {
			for key, value := range managedAnnotations {
				if currentValue, exists := toBeAnnotations[key]; exists && currentValue == value {
					delete(toBeAnnotations, key)
				}
			}
		}
		delete(toBeAnnotations, model.ManagedAnnotationsKey)
	}

	// add annotations by rules matched
	annotationsByRules := r.Matcher.GetAnnotationsForIngress(ingress)
	if len(annotationsByRules) > 0 {
		for key, value := range annotationsByRules {
			toBeAnnotations[key] = value
		}
		b := util.MustMarshalJSON(annotationsByRules)
		toBeAnnotations[model.ManagedAnnotationsKey] = string(b) + "\n"
	}
	return toBeAnnotations
}

func copyAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return make(map[string]string)
	}
	copy := make(map[string]string, len(annotations))
	for k, v := range annotations {
		copy[k] = v
	}
	return copy
}

func annotationsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

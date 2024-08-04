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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmnote/ingress-annotator/pkg/rulesstore"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	Client     client.Client
	Scheme     *runtime.Scheme
	RulesStore *rulesstore.RulesStore
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("namespace", req.Namespace, "name", req.Name)

	if req.Namespace != r.RulesStore.ConfigMapNamespace || req.Name != r.RulesStore.ConfigMapName {
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling ConfigMap")
	if err := r.RulesStore.UpdateData(); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update data in RulesStore: %w", err)
	}

	var ingressList networkingv1.IngressList
	if err := r.Client.List(ctx, &ingressList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list ingress resources: %w", err)
	}

	for _, ingress := range ingressList.Items {
		if ingress.Annotations[annotatorEnabledKey] == "true" {
			ingress.Annotations[annotatorReconcileNeededKey] = "true"
			if err := r.Client.Update(ctx, &ingress); err != nil {
				return ctrl.Result{}, fmt.Errorf("update ingress err: %w", err)
			}
		}
	}

	logger.Info("Successfully reconciled ConfigMap")
	return ctrl.Result{}, nil
}

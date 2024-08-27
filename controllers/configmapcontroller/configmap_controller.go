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

package configmapcontroller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	NN         types.NamespacedName
	RulesStore rulesstore.IRulesStore
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Only proceed if the request is for the ConfigMap we're watching
	if req.Namespace != r.NN.Namespace || req.Name != r.NN.Name {
		return ctrl.Result{}, nil
	}

	logger := ctrl.LoggerFrom(ctx).WithValues("kind", "ConfigMap", "namespace", req.Namespace, "name", req.Name)
	logger.Info("Reconciling ConfigMap")

	// Fetch the ConfigMap resource
	var cm corev1.ConfigMap
	if err := r.Get(ctx, r.NN, &cm); err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "ConfigMap %s not found, will retry after delay", r.NN)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update rules in the RulesStore
	oldRules := r.RulesStore.GetRules()
	logger.Info("Updating rules", "oldRules", oldRules)

	if err := r.RulesStore.UpdateRules(&cm); err != nil {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to update rules in rules store: %w", err)
	}

	newRules := r.RulesStore.GetRules()
	logger.Info("Rules updated", "newRules", newRules)

	logger.Info("Successfully reconciled ConfigMap")
	return ctrl.Result{}, nil
}

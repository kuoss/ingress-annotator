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
	"errors"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ConfigMeta types.NamespacedName
	RulesStore rulesstore.IRulesStore
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ns := os.Getenv("CONTROLLER_NAMESPACE")
	if ns == "" {
		return errors.New("CONTROLLER_NAMESPACE environment variable is not set or is empty")
	}
	r.ConfigMeta = types.NamespacedName{
		Namespace: ns,
		Name:      "ingress-annotator-rules",
	}
	if err := r.updateRulesWithConfigMap(context.Background()); err != nil {
		return fmt.Errorf("updateRulesWithConfigMap err: %w", err)
	}
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
	if req.Namespace == r.ConfigMeta.Namespace && req.Name == r.ConfigMeta.Name {
		return r.reconcileNormal(ctx, req)
	}
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) reconcileNormal(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("kind", "configmap", "namespace", req.Namespace, "name", req.Name).WithCallDepth(1)

	logger.Info("Reconciling ConfigMap")
	if err := r.updateRulesWithConfigMap(context.Background()); err != nil {
		return ctrl.Result{}, fmt.Errorf("updateRulesWithConfigMap err: %w", err)
	}

	logger.Info("Successfully reconciled ConfigMap")
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) updateRulesWithConfigMap(ctx context.Context) error {
	var cm corev1.ConfigMap
	if err := r.Get(ctx, r.ConfigMeta, &cm); err != nil {
		return fmt.Errorf("getConfigMap err: %w", err)
	}
	if err := r.RulesStore.UpdateRules(&cm); err != nil {
		return fmt.Errorf("failed to update data in rules store: %w", err)
	}
	return nil
}

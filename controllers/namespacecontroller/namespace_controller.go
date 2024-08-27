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

package namespacecontroller

import (
	"context"
	"fmt"

	"github.com/kuoss/ingress-annotator/pkg/model"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	IngressReconciler reconcile.Reconciler
	Recorder          record.EventRecorder
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx).WithValues("namespace", req.Namespace)

	namespace := &corev1.Namespace{}
	err := r.Client.Get(ctx, req.NamespacedName, namespace)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !namespace.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling Namespace")

	if err := r.annotateIngressesInNamespace(ctx, req.Namespace); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to annotate ingresses: %w", err)
	}

	logger.Info("Reconciled Namespace successfully")
	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) annotateIngressesInNamespace(ctx context.Context, namespace string) error {
	var ingressList networkingv1.IngressList

	if err := r.List(ctx, &ingressList, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ing := range ingressList.Items {
		if err := r.annotateIngress(ctx, ing); err != nil {
			return fmt.Errorf("failed to annotate ingress: %w", err)
		}
	}

	return nil
}

func (r *NamespaceReconciler) annotateIngress(ctx context.Context, ing networkingv1.Ingress) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, &ing); err != nil {
			return fmt.Errorf("failed to get latest ingress %s/%s: %w", ing.Namespace, ing.Name, err)
		}
		ing.SetAnnotations(map[string]string{model.ReconcileKey: "true"})
		if err := r.Update(ctx, &ing); err != nil {
			if apierrors.IsConflict(err) {
				return err
			}
			return fmt.Errorf("failed to update ingress %s/%s: %w", ing.Namespace, ing.Name, err)
		}
		return nil
	})
}

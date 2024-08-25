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
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

var (
	marshal = json.Marshal
)

type ingressScope struct {
	logger             logr.Logger
	namespace          *corev1.Namespace
	ingress            *networkingv1.Ingress
	updatedAnnotations model.Annotations
}

type IngressReconciler struct {
	client.Client
	RulesStore rulesstore.IRulesStore
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
	logger := ctrl.LoggerFrom(ctx).WithValues("ingress", req.NamespacedName)

	// Fetch Ingress resource
	var ingress networkingv1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Fetch Namespace resource
	var namespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: ingress.Namespace}, &namespace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize ingressScope
	scope := &ingressScope{
		logger:             logger,
		namespace:          &namespace,
		ingress:            &ingress,
		updatedAnnotations: copyAnnotations(ingress.Annotations), // Copy to avoid mutating original map
	}

	// Reconcile Ingress
	return r.reconcileIngress(ctx, scope)
}

func (r *IngressReconciler) reconcileIngress(ctx context.Context, scope *ingressScope) (ctrl.Result, error) {
	originalAnnotations := copyAnnotations(scope.updatedAnnotations)
	r.removeManagedAnnotations(scope)
	r.addNewAnnotations(scope)

	// Early exit if there are no changes to annotations.
	if annotationsEqual(originalAnnotations, scope.updatedAnnotations) {
		scope.logger.Info("No changes detected in annotations; skipping update")
		return ctrl.Result{}, nil
	}

	// Update the Ingress resource with new annotations.
	scope.ingress.Annotations = scope.updatedAnnotations
	if err := r.Update(ctx, scope.ingress); err != nil {
		scope.logger.Error(err, "Failed to update Ingress with new annotations")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	scope.logger.Info("Successfully reconciled Ingress with new annotations")
	return ctrl.Result{}, nil
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

func (r *IngressReconciler) removeManagedAnnotations(scope *ingressScope) {
	managedAnnotations := make(model.Annotations)
	if value, ok := scope.ingress.Annotations[model.ManagedAnnotationsKey]; ok && value != "" {
		if err := json.Unmarshal([]byte(value), &managedAnnotations); err != nil {
			scope.logger.Error(err, "Warning: Failed to unmarshal managed annotations")
		}
	}

	for key, value := range managedAnnotations {
		if currentValue, exists := scope.updatedAnnotations[key]; exists && currentValue == value {
			delete(scope.updatedAnnotations, key)
		}
	}
	delete(scope.updatedAnnotations, model.ReconcileKey)
	delete(scope.updatedAnnotations, model.ManagedAnnotationsKey)
}

func (r *IngressReconciler) addNewAnnotations(scope *ingressScope) {
	newAnnotations := r.getNewAnnotations(scope)
	for key, value := range newAnnotations {
		scope.updatedAnnotations[key] = value
	}
	if len(newAnnotations) == 0 {
		return
	}

	bytes, err := marshal(newAnnotations)
	if err != nil {
		scope.logger.Error(err, "Failed to marshal new annotations")
		return // Stop further processing if marshalling fails
	}
	scope.updatedAnnotations[model.ManagedAnnotationsKey] = string(bytes)
}

func (r *IngressReconciler) getNewAnnotations(scope *ingressScope) model.Annotations {
	ruleNames := r.getRuleNames(scope)
	rules := r.RulesStore.GetRules()
	newAnnotations := make(model.Annotations)

	for _, ruleName := range ruleNames {
		if annotations, exists := (*rules)[ruleName]; exists {
			for k, v := range annotations {
				newAnnotations[k] = v
			}
		} else {
			scope.logger.Info("Warning: no ruleName in rules", "ruleName", ruleName)
		}
	}
	return newAnnotations
}

func (r *IngressReconciler) getRuleNames(scope *ingressScope) []string {
	namespaceRuleNames := getRuleNamesFromObject(scope.namespace, model.RulesKey)
	ingressRuleNames := getRuleNamesFromObject(scope.ingress, model.RulesKey)
	return mergeRuleNames(namespaceRuleNames, ingressRuleNames)
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

func mergeRuleNames(names1, names2 []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, name := range names1 {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	for _, name := range names2 {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	return result
}

func getRuleNamesFromObject(obj client.Object, key string) []string {
	if value, ok := obj.GetAnnotations()[key]; ok && value != "" {
		names := strings.Split(value, ",")
		var cleaned []string
		for _, name := range names {
			if trimmedName := strings.TrimSpace(name); trimmedName != "" {
				cleaned = append(cleaned, trimmedName)
			}
		}
		return cleaned
	}
	return []string{}
}

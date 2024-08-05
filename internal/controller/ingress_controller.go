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
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/jmnote/ingress-annotator/pkg/rulesstore"
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RulesStore *rulesstore.RulesStore
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ingress networkingv1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ingress.Annotations[annotatorEnabledKey] != annotationEnabledValue {
		return ctrl.Result{}, nil
	}

	return r.reconcileAnnotations(ctx, &ingress)
}

func (r *IngressReconciler) reconcileAnnotations(ctx context.Context, ingress *networkingv1.Ingress) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", ingress.Namespace, "name", ingress.Name)
	logger.Info("Reconciling Ingress")

	if err := r.applyAnnotations(ctx, ingress); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to apply annotations to Ingress: %w", err)
	}

	logger.Info("Successfully reconciled Ingress")
	return ctrl.Result{}, nil

}

func (r *IngressReconciler) applyAnnotations(ctx context.Context, ingress *networkingv1.Ingress) error {
	data := r.RulesStore.GetData()

	if shouldSkipUpdate(ingress, data.ConfigMap.ResourceVersion) {
		return nil
	}

	lastAppliedRuleNames := parseCSVToSlice(ingress.Annotations[annotatorLastAppliedRulesKey])
	currentRuleNames := parseCSVToSlice(ingress.Annotations[annotatorRulesKey])
	deletedRuleNames := findDeletedRuleNames(lastAppliedRuleNames, currentRuleNames)

	removeAnnotations(ingress, deletedRuleNames, data.Rules)
	applyAnnotations(ingress, currentRuleNames, data.Rules)
	cleanupAnnotations(ingress, ingress.Annotations[annotatorRulesKey], data.ConfigMap.ResourceVersion)

	if err := r.Update(ctx, ingress); err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	return nil
}

func shouldSkipUpdate(ingress *networkingv1.Ingress, resourceVersion string) bool {
	return ingress.Annotations[annotatorReconcileNeededKey] != annotatorReconcileNeededValue &&
		ingress.Annotations[annotatorLastAppliedVersionKey] == resourceVersion
}

func removeAnnotations(ingress *networkingv1.Ingress, ruleNames []string, rules rulesstore.Rules) {
	for _, ruleName := range ruleNames {
		if annotations, exists := rules[ruleName]; exists {
			for key := range annotations {
				delete(ingress.Annotations, key)
			}
		}
	}
}

func applyAnnotations(ingress *networkingv1.Ingress, ruleNames []string, rules rulesstore.Rules) {
	for _, ruleName := range ruleNames {
		if annotations, exists := rules[ruleName]; exists {
			for key, value := range annotations {
				ingress.Annotations[key] = value
			}
		}
	}
}

func cleanupAnnotations(ingress *networkingv1.Ingress, currentRulesValue, resourceVersion string) {
	delete(ingress.Annotations, annotatorReconcileNeededKey)
	ingress.Annotations[annotatorLastAppliedRulesKey] = currentRulesValue
	ingress.Annotations[annotatorLastAppliedVersionKey] = resourceVersion
}

func parseCSVToSlice(csv string) []string {
	if csv == "" {
		return []string{}
	}
	return strings.Split(csv, ",")
}

func findDeletedRuleNames(lastApplied, current []string) []string {
	lastAppliedSet := make(map[string]struct{}, len(lastApplied))

	for _, rule := range lastApplied {
		lastAppliedSet[rule] = struct{}{}
	}

	for _, rule := range current {
		delete(lastAppliedSet, rule)
	}

	deleted := []string{}
	for rule := range lastAppliedSet {
		deleted = append(deleted, rule)
	}

	return deleted
}

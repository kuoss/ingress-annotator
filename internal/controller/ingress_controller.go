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
	"encoding/json"
	"fmt"
	"path/filepath"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

const (
	managedAnnotationsKey = "annotator.ingress.kubernetes.io/managed-annotations"
)

type IngressContext struct {
	ctx     context.Context
	logger  logr.Logger
	ingress networkingv1.Ingress
	rules   *model.Rules
}

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RulesStore rulesstore.IRulesStore
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

	ingressCtx := &IngressContext{
		ctx:     ctx,
		logger:  log.FromContext(ctx).WithValues("kind", "ingress", "namespace", ingress.Namespace, "name", ingress.Name).WithCallDepth(1),
		ingress: ingress,
		rules:   r.RulesStore.GetRules(),
	}

	newManagedAnnotations := r.getNewManagedAnnotations(ingressCtx)

	annotationsToRemove, warn := r.getAnnotationsToRemove(ingressCtx, newManagedAnnotations)
	if warn != nil {
		ingressCtx.logger.Info("failed to calculate annotations to remove: %v", warn)
	}

	annotationsToApply := r.getAnnotationsToApply(ingressCtx, newManagedAnnotations)

	// If no changes are required, return early
	if len(annotationsToRemove) == 0 && len(annotationsToApply) == 0 {
		return ctrl.Result{}, nil
	}

	// Handle annotation updates
	if err := r.updateAnnotations(ingressCtx, annotationsToRemove, annotationsToApply, newManagedAnnotations); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update ingress annotations: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) getNewManagedAnnotations(ctx *IngressContext) model.Annotations {
	ingress := ctx.ingress
	newManagedAnnotations := model.Annotations{}

	for key, rule := range *ctx.rules {
		if matched, err := filepath.Match(rule.Namespace, ingress.Namespace); err != nil {
			ctx.logger.Error(err, "failed to match namespace", "key", key, "namespace", rule.Namespace)
			continue
		} else if !matched {
			continue
		}

		if ingress.Name != "" {
			if matched, err := filepath.Match(rule.Ingress, ingress.Name); err != nil {
				ctx.logger.Error(err, "failed to match ingress name", "key", key, "ingress", rule.Ingress)
				continue
			} else if !matched {
				continue
			}
		}

		// Apply annotations from the matched rule
		for key, value := range rule.Annotations {
			newManagedAnnotations[key] = value
		}
	}

	return newManagedAnnotations
}

// updateAnnotations applies the calculated annotations to the Ingress resource.
func (r *IngressReconciler) updateAnnotations(ingressCtx *IngressContext, annotationsToRemove, annotationsToApply, newManagedAnnotations model.Annotations) error {
	ingress := ingressCtx.ingress

	for key := range annotationsToRemove {
		delete(ingress.Annotations, key)
	}

	for key, value := range annotationsToApply {
		ingress.Annotations[key] = value
	}

	newManagedAnnotationsBytes, err := json.Marshal(newManagedAnnotations)
	if err != nil {
		return fmt.Errorf("failed to marshal new managed annotations: %w", err) // test unreachable
	}

	ingress.Annotations[managedAnnotationsKey] = string(newManagedAnnotationsBytes)

	// Update the Ingress with the new annotations
	if err := r.Update(ingressCtx.ctx, &ingress); err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	ingressCtx.logger.Info("Successfully updated ingress annotations")
	return nil
}

func (r *IngressReconciler) getAnnotationsToRemove(ctx *IngressContext, newManagedAnnotations model.Annotations) (model.Annotations, error) {
	oldManagedAnnotationsValue, exists := ctx.ingress.Annotations[managedAnnotationsKey]
	if !exists {
		return nil, nil
	}

	oldManagedAnnotations := model.Annotations{}
	if err := json.Unmarshal([]byte(oldManagedAnnotationsValue), &oldManagedAnnotations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed annotations: %w", err)
	}

	annotationsToRemove := model.Annotations{}
	for key, value := range oldManagedAnnotations {
		// Remove only if the current value matches and it's not managed anymore
		if currentValue, exists := ctx.ingress.Annotations[key]; exists && currentValue == value {
			if _, exists := newManagedAnnotations[key]; !exists {
				annotationsToRemove[key] = value
			}
		}
	}
	return annotationsToRemove, nil
}

func (r *IngressReconciler) getAnnotationsToApply(ctx *IngressContext, newManagedAnnotations model.Annotations) model.Annotations {
	annotationsToApply := model.Annotations{}
	for key, value := range newManagedAnnotations {
		if currentValue, exists := ctx.ingress.Annotations[key]; !exists || currentValue != value {
			annotationsToApply[key] = value
		}
	}
	return annotationsToApply
}

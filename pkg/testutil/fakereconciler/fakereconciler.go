package fakereconciler

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type FakeReconciler struct {
}

func (r *FakeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

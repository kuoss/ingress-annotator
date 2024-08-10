package controller

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	return scheme
}

func newFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

type BadClient1 struct {
	client.Client
}

func (c *BadClient1) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return errors.New("Update operation is disabled in this fake client")
}

func newBadClient1(objs ...client.Object) client.Client {
	return &BadClient1{
		Client: newFakeClient(objs...),
	}
}

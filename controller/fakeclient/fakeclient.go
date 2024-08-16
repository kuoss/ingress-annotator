package fakeclient

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	return scheme
}

type ClientOpts struct {
	GetError    bool
	UpdateError bool
}

func NewClient(opts *ClientOpts, objs ...client.Object) client.Client {
	if opts == nil {
		opts = &ClientOpts{}
	}

	interceptorFuncs := interceptor.Funcs{}
	if opts.GetError {
		interceptorFuncs.Get = func(ctx context.Context, client client.WithWatch, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
			return errors.New("mocked Get error")
		}
	}
	if opts.UpdateError {
		interceptorFuncs.Update = func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			return errors.New("mocked Update error")
		}
	}
	return fake.NewClientBuilder().
		WithScheme(NewScheme()).
		WithInterceptorFuncs(interceptorFuncs).
		WithObjects(objs...).
		Build()
}

func NewManager() manager.Manager {
	mgr, err := ctrl.NewManager(&rest.Config{}, ctrl.Options{Scheme: NewScheme()})
	if err != nil {
		panic(err) // test unreachable
	}
	return mgr
}

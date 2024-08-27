package fakeclient

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func NewManager() manager.Manager {
	mgr, _ := ctrl.NewManager(&rest.Config{}, ctrl.Options{Scheme: NewScheme()})
	return mgr
}

type ClientOpts struct {
	GetError            bool
	GetNotFoundError    bool
	ListError           bool
	UpdateError         bool
	UpdateConflictError bool
}

func NewClient(opts *ClientOpts, objs ...client.Object) client.Client {
	interceptorFuncs := createInterceptorFuncs(opts)
	nonNilObjs := filterNonNilObjects(objs)

	return fake.NewClientBuilder().
		WithScheme(NewScheme()).
		WithInterceptorFuncs(interceptorFuncs).
		WithObjects(nonNilObjs...).
		Build()
}

func createInterceptorFuncs(opts *ClientOpts) interceptor.Funcs {
	if opts == nil {
		opts = &ClientOpts{}
	}

	funcs := interceptor.Funcs{}

	if opts.GetError {
		funcs.Get = func(
			ctx context.Context,
			client client.WithWatch,
			key types.NamespacedName,
			obj client.Object,
			opts ...client.GetOption,
		) error {
			return errors.New("mocked GetError")
		}
	}

	if opts.GetNotFoundError {
		funcs.Get = func(
			ctx context.Context,
			client client.WithWatch,
			key types.NamespacedName,
			obj client.Object,
			opts ...client.GetOption,
		) error {
			err := apierrors.NewNotFound(schema.GroupResource{Resource: "Resource"}, key.Name)
			return fmt.Errorf("mocked GetNotFoundError: %w", err)
		}
	}

	if opts.ListError {
		funcs.List = func(
			ctx context.Context,
			client client.WithWatch,
			list client.ObjectList,
			opts ...client.ListOption,
		) error {
			return errors.New("mocked ListError")
		}
	}

	if opts.UpdateError {
		funcs.Update = func(
			ctx context.Context,
			client client.WithWatch,
			obj client.Object,
			opts ...client.UpdateOption,
		) error {
			return errors.New("mocked UpdateError")
		}
	}
	if opts.UpdateConflictError {
		funcs.Update = func(
			ctx context.Context,
			client client.WithWatch,
			obj client.Object,
			opts ...client.UpdateOption,
		) error {
			err := apierrors.NewConflict(
				schema.GroupResource{Resource: "ingresses.networking.k8s.io"},
				obj.GetName(),
				errors.New("the object has been modified; please apply your changes to the latest version and try again"),
			)
			return fmt.Errorf("mocked UpdateConflictError: %w", err)
		}
	}

	return funcs
}

func filterNonNilObjects(objs []client.Object) []client.Object {
	nonNilObjs := make([]client.Object, 0, len(objs))
	for _, obj := range objs {
		if !reflect.ValueOf(obj).IsNil() {
			nonNilObjs = append(nonNilObjs, obj)
		}
	}
	return nonNilObjs
}

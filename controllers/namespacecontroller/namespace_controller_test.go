package namespacecontroller

import (
	"context"
	"testing"
	"time"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kuoss/ingress-annotator/pkg/testutil/fakeclient"
)

func TestNamespaceReconciler_SetupWithManager(t *testing.T) {
	client := fakeclient.NewClient(nil)
	reconciler := &NamespaceReconciler{
		Client: client,
	}

	err := reconciler.SetupWithManager(fakeclient.NewManager())
	assert.NoError(t, err)
}

func TestNamespaceReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, networkingv1.AddToScheme(scheme))

	tests := []struct {
		namespace  *corev1.Namespace
		clientOpts *fakeclient.ClientOpts
		wantResult ctrl.Result
		wantError  string
	}{
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			wantResult: ctrl.Result{},
		},
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace", DeletionTimestamp: &metav1.Time{Time: time.Now()}, Finalizers: []string{"test-finalizer"}}},
			wantResult: ctrl.Result{},
		},
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			clientOpts: &fakeclient.ClientOpts{NotFoundError: true},
			wantResult: ctrl.Result{},
		},
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			wantResult: ctrl.Result{},
			wantError:  "mocked Get error",
		},
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			clientOpts: &fakeclient.ClientOpts{ListError: true},
			wantResult: ctrl.Result{},
			wantError:  "failed to annotate ingresses: failed to list ingresses: mocked List error",
		},
		{
			namespace:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}},
			clientOpts: &fakeclient.ClientOpts{UpdateError: true},
			wantResult: ctrl.Result{},
			wantError:  "failed to annotate ingresses: failed to annotate ingress: failed to update ingress test-namespace/test-ingress: mocked Update error",
		},
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: ctrl.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-ingress",
		},
	}
	for i, tt := range tests {
		t.Run(testcase.Name(i, tt.namespace), func(t *testing.T) {
			client := fakeclient.NewClient(tt.clientOpts, tt.namespace, ingress)
			r := &NamespaceReconciler{
				Client: client,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}}
			result, err := r.Reconcile(context.Background(), req)

			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantError)
			}
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
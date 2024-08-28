package fakeclient

import (
	"context"
	"testing"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewClient_NilOpts(t *testing.T) {
	pod := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
	}
	cl := NewClient(nil, pod)
	gotPod := &networkingv1.Ingress{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: "test-ingress", Namespace: "default"}, gotPod)
	assert.NoError(t, err)
	assert.Equal(t, pod, gotPod)
	err = cl.Update(context.TODO(), pod)
	assert.NoError(t, err)
}

func TestNewClient_GetError(t *testing.T) {
	testCases := []struct {
		obj       client.Object
		getError  string
		wantError string
	}{
		{
			obj:       &networkingv1.Ingress{},
			getError:  "*",
			wantError: "mocked GetError",
		},
		{
			obj:       &corev1.ConfigMap{},
			getError:  "ConfigMap",
			wantError: "mocked GetError ConfigMap",
		},
		{
			obj:       &corev1.Namespace{},
			getError:  "Namespace",
			wantError: "mocked GetError Namespace",
		},
		{
			obj:       &networkingv1.Ingress{},
			getError:  "Ingress",
			wantError: "mocked GetError Ingress",
		},
		{
			obj:      &networkingv1.Ingress{},
			getError: "Namespace",
		},
		{
			obj:      &corev1.Node{},
			getError: "X",
		},
	}
	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.getError), func(t *testing.T) {
			opts := &ClientOpts{GetError: tc.getError}
			cl := NewClient(opts)
			err := cl.Get(context.TODO(), types.NamespacedName{}, tc.obj)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}

func TestNewClient_GetNotFoundError(t *testing.T) {
	wantError := `mocked GetNotFoundError: Resource "non-existent-pod" not found`
	opts := &ClientOpts{GetNotFoundError: true}
	cl := NewClient(opts)
	pod := &networkingv1.Ingress{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: "non-existent-pod", Namespace: "default"}, pod)
	assert.EqualError(t, err, wantError)
}

func TestNewClient_ListError(t *testing.T) {
	wantError := "mocked ListError"
	opts := &ClientOpts{ListError: true}
	cl := NewClient(opts)

	podList := &networkingv1.IngressList{}
	err := cl.List(context.TODO(), podList)
	assert.EqualError(t, err, wantError)
	assert.Empty(t, podList.Items)
}

func TestNewClient_UpdateError(t *testing.T) {
	wantError := "mocked UpdateError"
	pod := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
	}
	opts := &ClientOpts{UpdateError: true}
	cl := NewClient(opts, pod)
	err := cl.Update(context.TODO(), pod)
	assert.EqualError(t, err, wantError)
}

func TestNewClient_UpdateConflictError(t *testing.T) {
	wantError := `mocked UpdateConflictError: Operation cannot be fulfilled on ingresses.networking.k8s.io "test-ingress": the object has been modified; please apply your changes to the latest version and try again`
	pod := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
	}
	opts := &ClientOpts{UpdateConflictError: true}
	cl := NewClient(opts, pod)
	err := cl.Update(context.TODO(), pod)
	assert.EqualError(t, err, wantError)
}

func TestNewManager(t *testing.T) {
	got := NewManager()
	assert.NotEmpty(t, got)
}

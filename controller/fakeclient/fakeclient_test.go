package fakeclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewClient_NilOpts(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	cl := NewClient(nil, pod)
	gotPod := &corev1.Pod{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: "test-pod", Namespace: "default"}, gotPod)
	assert.NoError(t, err)
	assert.Equal(t, pod, gotPod)
	err = cl.Update(context.TODO(), pod)
	assert.NoError(t, err)
}

func TestNewClient_GetError(t *testing.T) {
	opts := &ClientOpts{GetError: true}
	cl := NewClient(opts)
	pod := &corev1.Pod{}
	err := cl.Get(context.TODO(), types.NamespacedName{Name: "test-pod", Namespace: "default"}, pod)
	assert.EqualError(t, err, "mocked Get error")
}

func TestNewClient_UpdateError(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	opts := &ClientOpts{UpdateError: true}
	cl := NewClient(opts, pod)
	err := cl.Update(context.TODO(), pod)
	assert.EqualError(t, err, "mocked Update error")
}

func TestNewManager(t *testing.T) {
	got := NewManager()
	assert.NotEmpty(t, got)
}

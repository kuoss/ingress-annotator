package configmapcontroller

import (
	"context"
	"testing"
	"time"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
	"github.com/kuoss/ingress-annotator/pkg/testutil/fakeclient"
)

func TestConfigMapReconciler_SetupWithManager(t *testing.T) {
	testCases := []struct {
		name      string
		mgr       ctrl.Manager
		wantError string
	}{
		{
			name:      "nil Manager should return error",
			mgr:       nil,
			wantError: "must provide a non-nil Manager",
		},
		{
			name:      "valid Manager should not return error",
			mgr:       fakeclient.NewManager(),
			wantError: "",
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			reconciler := &ConfigMapReconciler{}
			err := reconciler.SetupWithManager(tc.mgr)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

func TestConfigMapReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Helper function to create a ConfigMap
	createConfigMap := func(namespace, name, rulesText string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: ctrl.ObjectMeta{Namespace: namespace, Name: name},
			Data:       map[string]string{"rules": rulesText},
		}
	}

	testCases := []struct {
		name       string
		clientOpts *fakeclient.ClientOpts
		cm         *corev1.ConfigMap
		newCM      *corev1.ConfigMap
		nn         types.NamespacedName
		requestNN  types.NamespacedName
		want       ctrl.Result
		wantError  string
	}{
		{
			name:       "Error on ConfigMap Get should return appropriate error and requeue",
			clientOpts: &fakeclient.ClientOpts{GetError: true}, // Simulate a Get error on Get
			cm:         createConfigMap("default", "ingress-annotator", ""),
			nn:         types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			wantError:  "failed to get ConfigMap: mocked Get error",
			want:       ctrl.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name:       "ConfigMap not found should requeue after 30 seconds",
			clientOpts: &fakeclient.ClientOpts{NotFoundError: true}, // Simulate a NotFound error on Get
			cm:         createConfigMap("default", "ingress-annotator", ""),
			nn:         types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:       ctrl.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name:      "Invalid ConfigMap data should return unmarshalling error",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			newCM:     createConfigMap("default", "ingress-annotator", "invalid rules"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError: "failed to update rules in rules store: failed to extract rules from configMap: failed to unmarshal rules: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rules",
		},
		{
			name:      "Valid ConfigMap but no change should not requeue or return an error",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			newCM:     createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{},
		},
		{
			name:      "Valid ConfigMap should process without errors or requeue",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{},
		},
		{
			name:      "Valid ConfigMap but different request name should not return errors",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "xxx"},
			want:      ctrl.Result{},
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {

			ctx := context.Background()

			client := fakeclient.NewClient(tc.clientOpts, tc.cm)
			store, err := rulesstore.New(tc.cm)
			assert.NoError(t, err)
			reconciler := &ConfigMapReconciler{
				NN:         tc.nn,
				Client:     client,
				RulesStore: store,
			}

			if tc.newCM != nil {
				err := client.Update(ctx, tc.newCM)
				assert.NoError(t, err)
			}

			got, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: tc.requestNN})
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

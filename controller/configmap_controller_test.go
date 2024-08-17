package controller

import (
	"context"
	"testing"

	"github.com/jmnote/tester"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
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
	// Helper function to create a ConfigMap
	createConfigMap := func(namespace, name, data string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: ctrl.ObjectMeta{Namespace: namespace, Name: name},
			Data:       map[string]string{"rule1": data},
		}
	}

	testCases := []struct {
		name       string
		clientOpts *fakeclient.ClientOpts
		cm         *corev1.ConfigMap
		newCM      *corev1.ConfigMap
		requestNN  types.NamespacedName
		want       ctrl.Result
		wantError  string
	}{
		{
			name:       "client Get error should return an error",
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			cm:         createConfigMap("default", "ingress-annotator", "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"),
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			wantError:  "failed to get ConfigMap: mocked Get error",
		},
		{
			name:      "invalid ConfigMap data should return an error",
			cm:        createConfigMap("default", "ingress-annotator", "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"),
			newCM:     createConfigMap("default", "ingress-annotator", "invalid data"),
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			wantError: "failed to update rules in rules store: invalid data in ConfigMap key rule1: failed to unmarshal YAML: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rule",
		},
		{
			name:      "valid ConfigMap should not return an error",
			cm:        createConfigMap("default", "ingress-annotator", "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"),
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      reconcile.Result{},
		},
		{
			name:      "ConfigMap with non-matching name should not return an error",
			cm:        createConfigMap("default", "ingress-annotator", "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"),
			requestNN: types.NamespacedName{Namespace: "default", Name: "xxx"},
			want:      reconcile.Result{Requeue: false, RequeueAfter: 10000000000},
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()

			client := fakeclient.NewClient(tc.clientOpts, tc.cm)
			store, err := rulesstore.New(tc.cm)
			assert.NoError(t, err)

			reconciler := &ConfigMapReconciler{
				NN:         tc.requestNN,
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

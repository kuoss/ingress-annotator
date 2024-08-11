package controller

import (
	"context"
	"testing"

	"github.com/jmnote/tester"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

func TestConfigMapReconciler_SetupWithManager(t *testing.T) {
	testCases := []struct {
		name         string
		namespaceEnv string
		objects      []client.Object
		wantError    string
	}{
		{
			name:         "successful setup 1",
			namespaceEnv: "default",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "ingress-annotator-rules",
						Namespace: "default",
					},
				},
			},
			wantError: "",
		},
		{
			name:         "successful setup 2",
			namespaceEnv: "default",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "ingress-annotator-rules",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": "annotations:\n  key1: value1\nnamespace: test-namespace\ningress: test-ingress",
					},
				},
			},
			wantError: "",
		},
		{
			name:         "missing CONTROLLER_NAMESPACE",
			namespaceEnv: "",
			objects:      []client.Object{},
			wantError:    "CONTROLLER_NAMESPACE environment variable is not set or is empty",
		},
		{
			name:         "successful setup",
			namespaceEnv: "default",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "ingress-annotator-rules",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": "invalid",
					},
				},
			},
			wantError: "yaml: unmarshal errors",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			t.Setenv("CONTROLLER_NAMESPACE", tc.namespaceEnv)

			reconciler := &ConfigMapReconciler{
				Client:     newFakeClient(tc.objects...),
				Scheme:     newScheme(),
				RulesStore: rulesstore.New(),
			}

			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme: newScheme(),
			})
			assert.NoError(t, err)

			err = reconciler.SetupWithManager(mgr)
			if tc.wantError != "" {
				assert.ErrorContains(t, err, tc.wantError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestConfigMapReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name        string
		configMeta  types.NamespacedName
		objects     []client.Object
		requestMeta types.NamespacedName
		wantResult  reconcile.Result
		wantError   string
	}{
		{
			name:       "successful reconciliation",
			configMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{Namespace: "default", Name: "ingress-annotator-rules"},
					Data:       map[string]string{"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"},
				},
			},
			requestMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
			wantResult:  reconcile.Result{},
			wantError:   "",
		},
		{
			name:       "successful reconciliation for other cm",
			configMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{Namespace: "default", Name: "ingress-annotator-rules"},
					Data:       map[string]string{"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"},
				},
			},
			requestMeta: types.NamespacedName{Namespace: "default", Name: "xxx"},
			wantResult:  reconcile.Result{},
			wantError:   "",
		},
		{
			name:        "config map not found",
			configMeta:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
			objects:     []client.Object{},
			requestMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
			wantResult:  reconcile.Result{},
			wantError:   `updateRulesWithConfigMap err: getConfigMap err: configmaps "ingress-annotator-rules" not found`,
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()
			client := newFakeClient(tc.objects...)
			store := rulesstore.New()
			reconciler := &ConfigMapReconciler{
				Client:     client,
				Scheme:     runtime.NewScheme(),
				ConfigMeta: tc.configMeta,
				RulesStore: store,
			}

			req := ctrl.Request{NamespacedName: tc.requestMeta}
			result, err := reconciler.Reconcile(ctx, req)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
			assert.Equal(t, tc.wantResult, result)
		})
	}
}

func TestConfigMapReconciler_updateRulesWithConfigMap(t *testing.T) {
	testCases := []struct {
		name          string
		configMap     *corev1.ConfigMap
		expectedError bool
	}{
		{
			name: "valid config map",
			configMap: &corev1.ConfigMap{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "ingress-annotator-rules",
					Namespace: "default",
				},
				Data: map[string]string{
					"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress",
				},
			},
			expectedError: false,
		},
		{
			name: "invalid config map data",
			configMap: &corev1.ConfigMap{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      "ingress-annotator-rules",
					Namespace: "default",
				},
				Data: map[string]string{
					"rule1": "invalid yaml",
				},
			},
			expectedError: true,
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			client := newFakeClient(tc.configMap)
			store := rulesstore.New()
			reconciler := &ConfigMapReconciler{
				Client:     client,
				Scheme:     runtime.NewScheme(),
				ConfigMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator-rules"},
				RulesStore: store,
			}

			err := reconciler.updateRulesWithConfigMap(context.Background())
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				rules := store.GetRules()
				assert.NotNil(t, rules)
				assert.Contains(t, *rules, "rule1")
			}
		})
	}
}

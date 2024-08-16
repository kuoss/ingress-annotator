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

	"github.com/kuoss/ingress-annotator/controller/fakeclient"
	"github.com/kuoss/ingress-annotator/controller/rulesstore"
)

func TestConfigMapReconciler_SetupWithManager(t *testing.T) {
	testCases := []struct {
		name         string
		namespaceEnv string
		objects      []client.Object
		wantError    string
	}{
		{
			name: "successful setup 1",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
				},
			},
			wantError: "",
		},
		{
			name: "successful setup 2",
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": "annotations:\n  key1: value1\nnamespace: test-namespace\ningress: test-ingress",
					},
				},
			},
			wantError: "",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			nn := types.NamespacedName{Namespace: "default", Name: "ingress-annotator"}
			client := fakeclient.NewClient(nil, tc.objects...)
			store, err := rulesstore.New(client, nn)
			assert.NoError(t, err)
			reconciler := &ConfigMapReconciler{
				Client:     client,
				Scheme:     fakeclient.NewScheme(),
				RulesStore: store,
			}
			err = reconciler.SetupWithManager(fakeclient.NewManager())
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}
func TestConfigMapReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name        string
		configNN    types.NamespacedName
		objects     []client.Object
		requestMeta types.NamespacedName
		wantResult  reconcile.Result
		wantError   string
	}{
		{
			name:     "successful reconciliation",
			configNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{Namespace: "default", Name: "ingress-annotator"},
					Data:       map[string]string{"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"},
				},
			},
			requestMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			wantResult:  reconcile.Result{},
			wantError:   "",
		},
		{
			name:     "successful reconciliation for other cm",
			configNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: ctrl.ObjectMeta{Namespace: "default", Name: "ingress-annotator"},
					Data:       map[string]string{"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress"},
				},
			},
			requestMeta: types.NamespacedName{Namespace: "default", Name: "xxx"},
			wantResult:  reconcile.Result{},
			wantError:   "",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()
			nn := types.NamespacedName{Namespace: "default", Name: "ingress-annotator"}
			client := fakeclient.NewClient(nil, tc.objects...)
			store, err := rulesstore.New(client, nn)
			assert.NoError(t, err)
			reconciler := &ConfigMapReconciler{
				Client:     client,
				Scheme:     runtime.NewScheme(),
				ConfigNN:   tc.configNN,
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

// func TestConfigMapReconciler_updateRulesWithConfigMap(t *testing.T) {
// 	testCases := []struct {
// 		name          string
// 		configMap     *corev1.ConfigMap
// 		expectedError bool
// 	}{
// 		{
// 			name: "valid config map",
// 			configMap: &corev1.ConfigMap{
// 				ObjectMeta: ctrl.ObjectMeta{
// 					Name:      "ingress-annotator",
// 					Namespace: "default",
// 				},
// 				Data: map[string]string{
// 					"rule1": "annotations:\n  key: value\nnamespace: default\ningress: my-ingress",
// 				},
// 			},
// 			expectedError: false,
// 		},
// 		{
// 			name: "invalid config map data",
// 			configMap: &corev1.ConfigMap{
// 				ObjectMeta: ctrl.ObjectMeta{
// 					Name:      "ingress-annotator",
// 					Namespace: "default",
// 				},
// 				Data: map[string]string{
// 					"rule1": "invalid yaml",
// 				},
// 			},
// 			expectedError: true,
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
// 			client := newFakeClient(nil, tc.configMap)
// 			store, err := rulesstore.New()
// 			assert.NoError(t, err)
// 			reconciler := &ConfigMapReconciler{
// 				Client:     client,
// 				Scheme:     runtime.NewScheme(),
// 				ConfigMeta: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
// 				RulesStore: store,
// 			}

// 			err = reconciler.updateRulesWithConfigMap(context.Background())
// 			if tc.expectedError {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 				rules := store.GetRules()
// 				assert.NotNil(t, rules)
// 				assert.Contains(t, *rules, "rule1")
// 			}
// 		})
// 	}
// }

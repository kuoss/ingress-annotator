package controller

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/jmnote/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/kuoss/ingress-annotator/pkg/model"
)

// TestSetupWithManager tests the SetupWithManager method of IngressReconciler.
func TestSetupWithManager(t *testing.T) {
	mockRulesStore := new(MockRulesStore)
	rules := &model.Rules{
		"default/example-ingress": {
			Namespace: "default",
			Ingress:   "example-ingress",
			Annotations: map[string]string{
				"new-key": "new-value",
			},
		},
	}
	mockRulesStore.On("GetRules").Return(rules)

	reconciler := &IngressReconciler{
		Client:     newFakeClient(),
		Scheme:     newScheme(),
		RulesStore: mockRulesStore,
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: newScheme(),
	})
	assert.NoError(t, err)

	err = reconciler.SetupWithManager(mgr)
	assert.NoError(t, err)
}

// Mocked RulesStore for testing
type MockRulesStore struct {
	mock.Mock
	Rules *model.Rules
}

func (m *MockRulesStore) GetRules() *model.Rules {
	args := m.Called()
	return args.Get(0).(*model.Rules)
}

func (m *MockRulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	args := m.Called()
	return args.Get(0).(error)
}

// TestReconcile tests the Reconcile method of IngressReconciler
func TestReconcile(t *testing.T) {
	testCases := []struct {
		name        string
		ingress     *networkingv1.Ingress
		rules       *model.Rules
		badClient1  bool
		wantApplied map[string]string
		wantRemoved []string
		wantError   string
	}{
		{
			name: "should apply new annotations",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"example-key": "example-value",
					},
				},
			},
			rules: &model.Rules{
				"default/example-ingress": {
					Namespace: "default",
					Ingress:   "example-ingress",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			wantApplied: map[string]string{"new-key": "new-value"},
			wantRemoved: nil,
			wantError:   "",
		},
		{
			name: "should remove outdated annotations",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"annotator.ingress.kubernetes.io/managed-annotations": `{"outdated-key":"outdated-value"}`,
						"outdated-key": "outdated-value",
					},
				},
			},
			rules: &model.Rules{
				"default/example-ingress": {
					Namespace: "default",
					Ingress:   "example-ingress",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			wantApplied: map[string]string{"new-key": "new-value"},
			wantRemoved: []string{"outdated-key"},
			wantError:   "",
		},
		{
			name: "should return early if no changes are required",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			rules: &model.Rules{
				"default/example-ingress": {
					Namespace: "default",
					Ingress:   "example-ingress",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			wantApplied: nil,
			wantRemoved: nil,
			wantError:   "",
		},
		{
			name: "invalid JSON in managed annotations",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
						"new-key": "new-value",
					},
				},
			},
			rules: &model.Rules{
				"default/example-ingress": {
					Namespace: "default",
					Ingress:   "example-ingress",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			wantApplied: nil,
			wantRemoved: nil,
			wantError:   "",
		},
		{
			name:        "should handle ingress not found",
			ingress:     nil,
			rules:       &model.Rules{},
			wantApplied: nil,
			wantRemoved: nil,
			wantError:   "",
		},
		{
			name: "Return error when client fails to update annotations",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-ingress",
					Namespace: "default",
					Annotations: map[string]string{
						"example-key": "example-value",
					},
				},
			},
			rules: &model.Rules{
				"default/example-ingress": {
					Namespace: "default",
					Ingress:   "example-ingress",
					Annotations: map[string]string{
						"new-key": "new-value",
					},
				},
			},
			wantApplied: map[string]string{"new-key": "new-value"},
			wantRemoved: nil,
			badClient1:  true,
			wantError:   "failed to update ingress annotations",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			objects := []client.Object{}
			if tc.ingress != nil {
				objects = []client.Object{tc.ingress}
			}
			client := newFakeClient(objects...)
			if tc.badClient1 {
				client = newBadClient1(objects...)
			}
			// Setup the IngressReconciler with a mock RulesStore
			mockRulesStore := new(MockRulesStore)
			mockRulesStore.On("GetRules").Return(tc.rules)

			reconciler := &IngressReconciler{
				Client:     client,
				Scheme:     newScheme(),
				RulesStore: mockRulesStore,
			}

			// Run Reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "example-ingress",
					Namespace: "default",
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			if tc.wantError != "" {
				assert.ErrorContains(t, err, tc.wantError)
				return
			} else {
				assert.NoError(t, err)
			}

			// If ingress was found, retrieve and check the updated Ingress
			if tc.ingress != nil {
				updatedIngress := &networkingv1.Ingress{}
				err = client.Get(context.Background(), req.NamespacedName, updatedIngress)
				assert.NoError(t, err)

				// Check that the new annotations were applied
				for key, value := range tc.wantApplied {
					assert.Equal(t, value, updatedIngress.Annotations[key])
				}

				// Check that the removed annotations are no longer present
				for _, key := range tc.wantRemoved {
					_, exists := updatedIngress.Annotations[key]
					assert.False(t, exists)
				}
			}
		})
	}
}

func TestGetManagedAnnotations(t *testing.T) {
	testCases := []struct {
		rules   model.Rules
		ingress networkingv1.Ingress
		want    model.Annotations
	}{
		{
			rules: map[string]model.Rule{
				"rule1": {
					Namespace:   "default",
					Ingress:     "example-ingress",
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "example-ingress",
				},
			},
			want: map[string]string{"key1": "value1"},
		},
		{
			rules: map[string]model.Rule{
				"rule1": {
					Namespace:   "xxx",
					Ingress:     "example-ingress",
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "example-ingress",
				},
			},
			want: map[string]string{},
		},
		{
			rules: map[string]model.Rule{
				"rule1": {
					Namespace:   "default",
					Ingress:     "xxx",
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "example-ingress",
				},
			},
			want: map[string]string{},
		},
		{
			rules: map[string]model.Rule{
				"rule1": {
					Namespace:   "[",
					Ingress:     "example-ingress",
					Annotations: map[string]string{"key1": "value1"},
				},
			},
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "example-ingress",
				},
			},
			want: map[string]string{},
		},
		{
			rules: map[string]model.Rule{
				"rule1": {
					Namespace:   "default",
					Ingress:     "[",
					Annotations: map[string]string{"key2": "value2"},
				},
			},
			ingress: networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "example-ingress",
				},
			},
			want: map[string]string{},
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.rules), func(t *testing.T) {
			ctx := &IngressContext{
				logger:  zap.New(zap.UseDevMode(true)),
				ingress: tc.ingress,
				rules:   &tc.rules,
			}

			r := &IngressReconciler{
				Client: newFakeClient(&tc.ingress),
				Scheme: newScheme(),
			}

			got := r.getNewManagedAnnotations(ctx)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUpdateAnnotations(t *testing.T) {

	testCases := []struct {
		name                  string
		initialAnnotations    map[string]string
		annotationsToRemove   map[string]string
		annotationsToApply    map[string]string
		newManagedAnnotations map[string]string
		wantResult            map[string]string
		badClient1            bool
		wantError             string
	}{
		{
			name: "Remove and apply annotations successfully",
			initialAnnotations: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			annotationsToRemove:   map[string]string{"annotation1": ""},
			annotationsToApply:    map[string]string{"annotation3": "value3"},
			newManagedAnnotations: map[string]string{"annotation3": "value3"},
			wantResult: map[string]string{
				"annotation2": "value2",
				"annotation3": "value3",
			},
			wantError: "",
		},
		{
			name: "Apply annotations without removing any",
			initialAnnotations: map[string]string{
				"annotation1": "value1",
			},
			annotationsToRemove:   map[string]string{},
			annotationsToApply:    map[string]string{"annotation2": "value2"},
			newManagedAnnotations: map[string]string{"annotation2": "value2"},
			wantResult: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			wantError: "",
		},
		{
			name: "Remove all annotations",
			initialAnnotations: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			annotationsToRemove: map[string]string{
				"annotation1": "",
				"annotation2": "",
			},
			annotationsToApply:    map[string]string{},
			newManagedAnnotations: map[string]string{},
			wantResult:            map[string]string{},
			wantError:             "",
		},
		{
			name: "Remove all annotations",
			initialAnnotations: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			annotationsToRemove: map[string]string{
				"annotation1": "",
				"annotation2": "",
			},
			annotationsToApply:    map[string]string{},
			newManagedAnnotations: map[string]string{"": ""},
			wantResult:            map[string]string{},
			wantError:             "",
		},
		{
			name: "Bad client",
			initialAnnotations: map[string]string{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			annotationsToRemove: map[string]string{
				"annotation1": "",
				"annotation2": "",
			},
			annotationsToApply:    map[string]string{},
			newManagedAnnotations: map[string]string{},
			wantResult:            map[string]string{},
			badClient1:            true,
			wantError:             "failed to update ingress: Update operation is disabled in this fake client",
		},
	}

	// Iterate over test cases
	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			ingressName := "test-ingress"
			ingressNamespace := "default"

			// Create a fake ingress resource with initial annotations
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ingressName,
					Namespace:   ingressNamespace,
					Annotations: tc.initialAnnotations,
				},
			}

			// Create a fake client and IngressContext
			client := newFakeClient(ingress)
			if tc.badClient1 {
				client = newBadClient1(ingress)
			}

			logger := logr.Discard() // Using discard logger for testing

			ingressCtx := &IngressContext{
				ctx:     context.TODO(),
				ingress: *ingress,
				logger:  logger,
			}

			// Initialize the reconciler
			r := &IngressReconciler{
				Client: client,
			}

			// Call the function
			err := r.updateAnnotations(ingressCtx, tc.annotationsToRemove, tc.annotationsToApply, tc.newManagedAnnotations)

			// Check for errors if expected
			if tc.wantError != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantError)
				return
			} else {
				assert.NoError(t, err)
			}

			// Fetch the updated ingress
			updatedIngress := &networkingv1.Ingress{}
			err = client.Get(context.TODO(), types.NamespacedName{Name: ingressName, Namespace: ingressNamespace}, updatedIngress)
			assert.NoError(t, err)

			// Assert that the annotations match the expected result
			for key, value := range tc.wantResult {
				assert.Equal(t, value, updatedIngress.Annotations[key])
			}

			for key := range tc.annotationsToRemove {
				assert.NotContains(t, updatedIngress.Annotations, key)
			}

			// Verify managed annotations are correctly set
			newManagedAnnotationsBytes, err := json.Marshal(tc.newManagedAnnotations)
			assert.NoError(t, err)
			assert.Equal(t, string(newManagedAnnotationsBytes), updatedIngress.Annotations["annotator.ingress.kubernetes.io/managed-annotations"])
		})
	}
}

func TestGetAnnotationsToRemove(t *testing.T) {
	testCases := []struct {
		name               string
		ingressAnnotations model.Annotations
		managedAnnotations model.Annotations
		wantResult         model.Annotations
		wantError          string
	}{
		{
			name:               "No managed annotations exist",
			ingressAnnotations: model.Annotations{},
			managedAnnotations: model.Annotations{
				"example.com/key1": "value1",
			},
			wantResult: nil,
			wantError:  "",
		},
		{
			name: "Managed annotations exist and match, but not managed anymore",
			ingressAnnotations: model.Annotations{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"example.com/key1":"value1"}`,
				"example.com/key1": "value1",
			},
			managedAnnotations: model.Annotations{
				"example.com/key2": "value2",
			},
			wantResult: model.Annotations{
				"example.com/key1": "value1",
			},
			wantError: "",
		},
		{
			name: "Managed annotations exist but new managed",
			ingressAnnotations: model.Annotations{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"example.com/key1":"value1"}`,
				"example.com/key1": "value1",
			},
			managedAnnotations: model.Annotations{
				"example.com/key1": "value1",
			},
			wantResult: model.Annotations{},
			wantError:  "",
		},
		{
			name: "Managed annotations do not match current value",
			ingressAnnotations: model.Annotations{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"example.com/key1":"value1"}`,
				"example.com/key1": "different_value",
			},
			managedAnnotations: model.Annotations{},
			wantResult:         model.Annotations{},
			wantError:          "",
		},
		{
			name: "Invalid JSON in managed annotations",
			ingressAnnotations: model.Annotations{
				"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
			},
			managedAnnotations: model.Annotations{},
			wantResult:         nil,
			wantError:          "failed to unmarshal managed annotations: invalid character 'i' looking for beginning of value",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			ingress := networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tc.ingressAnnotations,
				},
			}

			ctx := &IngressContext{
				ingress: ingress,
			}

			client := fake.NewClientBuilder().WithObjects(&ingress).Build()

			r := &IngressReconciler{
				Client: client,
			}
			result, err := r.getAnnotationsToRemove(ctx, tc.managedAnnotations)

			if tc.wantError != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantError)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantResult, result)
		})
	}
}

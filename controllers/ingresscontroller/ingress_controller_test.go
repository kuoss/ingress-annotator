package ingresscontroller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/testutil/fakeclient"
	"github.com/kuoss/ingress-annotator/pkg/testutil/mocks"
)

func TestIngressReconciler_SetupWithManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	store := mocks.NewMockIRulesStore(mockCtrl)

	client := fakeclient.NewClient(nil)
	reconciler := &IngressReconciler{
		Client:     client,
		RulesStore: store,
	}

	err := reconciler.SetupWithManager(fakeclient.NewManager())
	assert.NoError(t, err)
}

func createIngress(namespace, name string, annotations map[string]string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: ctrl.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Annotations: annotations,
		},
	}
}

func createNamespace(name string, annotations map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: ctrl.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
	}
}

func TestIngressReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		name            string
		clientOpts      *fakeclient.ClientOpts
		namespace       *corev1.Namespace
		ingress         *networkingv1.Ingress
		rules           *model.Rules
		requestNN       *types.NamespacedName
		wantResult      ctrl.Result
		wantAnnotations map[string]string
		wantError       string
		wantGetError    string
	}{
		{
			name:         "IngressNotFound_ShouldReturnNotFoundError",
			wantResult:   ctrl.Result{},
			wantGetError: `ingresses.networking.k8s.io "my-ingress" not found`,
		},
		{
			name: "RulesProvidedButIngressNotFound_ShouldReturnNotFoundError",
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult:   ctrl.Result{},
			wantGetError: `ingresses.networking.k8s.io "my-ingress" not found`,
		},
		{
			name: "InvalidManagedAnnotations_ShouldRetainInvalidAnnotations",
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
				"example-key": "example-value",
			}),
			rules: &model.Rules{
				"rule": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
				"example-key": "example-value",
			},
		},
		{
			name:      "InvalidAnnotationsWithNamespace_ShouldResetInvalidAnnotations",
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
				"example-key": "example-value",
			}),
			rules: &model.Rules{
				"default/my-ingress": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{}",
				"example-key": "example-value",
			},
		},
		{
			name:      "ValidIngress_ShouldAddNewAnnotations",
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/rules": "rule1",
			}),
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
				"new-key":                                             "new-value",
			},
		},
		{
			name:      "ValidIngressWithMatchingRule_ShouldAddAnnotations",
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/rules": "rule1",
			}),
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
				"new-key":                                             "new-value",
			},
		},
		{
			name:      "ValidIngressWithMatchingRule_ShouldRetainAnnotations",
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
			}),
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
				"new-key":                                             "new-value",
			},
		},
		{
			name:      "ValidIngressWithUnmatchingRule_ShouldRetainExistingAnnotations",
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/rules": "xxx",
			}),
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{}`,
				"annotator.ingress.kubernetes.io/rules":               "xxx",
			},
		},
		{
			name: "NoChangesDetected_ShouldReturnEarlyWithoutUpdates",
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			namespace: createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
				"new-key":                                             "new-value",
			}),
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
				"new-key":                                             "new-value",
			},
		},
		{
			name:       "ClientGetError_ShouldReturnError",
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			namespace:  createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"example-key": "example-value",
			}),
			wantResult: ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError:  "mocked Get error",
		},
		{
			name:       "ClientUpdateError_ShouldRequeueAfterError",
			clientOpts: &fakeclient.ClientOpts{UpdateError: true},
			namespace:  createNamespace("default", nil),
			ingress: createIngress("default", "my-ingress", map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": `{"new-key":"new-value"}`,
				"annotator.ingress.kubernetes.io/rules":               "rule1",
			}),
			rules: &model.Rules{
				"rule1": {
					"new-key": "new-value",
				},
			},
			wantResult: ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError:  "mocked Update error",
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()
			nn := types.NamespacedName{Namespace: "default", Name: "my-ingress"}
			if tc.requestNN != nil {
				nn = *tc.requestNN
			}

			// Create the fake client with the ingress and namespace
			client := fakeclient.NewClient(tc.clientOpts, tc.ingress, tc.namespace)

			// Mock the rules store
			store := mocks.NewMockIRulesStore(mockCtrl)
			store.EXPECT().GetRules().Return(tc.rules).AnyTimes()

			reconciler := &IngressReconciler{
				Client:     client,
				RulesStore: store,
			}

			// Run the Reconcile method
			got, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nn})

			assert.Equal(t, tc.wantResult, got)

			if tc.wantError != "" {
				assert.EqualError(t, err, tc.wantError)
				return
			} else {
				assert.NoError(t, err)
			}

			updatedIngress := &networkingv1.Ingress{}
			err = client.Get(ctx, nn, updatedIngress)
			if tc.wantGetError != "" {
				assert.EqualError(t, err, tc.wantGetError)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.wantAnnotations, updatedIngress.Annotations)
		})
	}
}

func TestCopyAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]string
		expectedOutput map[string]string
		modifyCopy     bool
	}{
		{
			name:           "Nil Input",
			input:          nil,
			expectedOutput: map[string]string{},
		},
		{
			name:           "Empty Map",
			input:          map[string]string{},
			expectedOutput: map[string]string{},
		},
		{
			name: "Single Pair",
			input: map[string]string{
				"key1": "value1",
			},
			expectedOutput: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "Multiple Pairs",
			input: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expectedOutput: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "Original Map Unmodified",
			input: map[string]string{
				"key1": "value1",
			},
			expectedOutput: map[string]string{
				"key1": "value1",
			},
			modifyCopy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy := copyAnnotations(tt.input)
			assert.Equal(t, tt.expectedOutput, copy)

			if tt.modifyCopy {
				copy["key1"] = "modifiedValue"
				assert.Equal(t, "value1", tt.input["key1"], "Original map should not be modified")
			}
		})
	}
}

func TestIngressReconciler_addNewAnnotations_MarshalError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	originalMarshal := marshal
	defer func() {
		marshal = originalMarshal
	}()
	marshal = func(v interface{}) ([]byte, error) {
		return nil, errors.New("mock marshalling error")
	}

	store := mocks.NewMockIRulesStore(mockCtrl)
	store.EXPECT().GetRules().Return(&model.Rules{}).AnyTimes()
	reconciler := &IngressReconciler{
		RulesStore: store,
	}

	scope := &ingressScope{
		namespace:          &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}},
		ingress:            &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}}},
		updatedAnnotations: map[string]string{},
	}
	reconciler.addNewAnnotations(scope)
}

func TestGetRuleNamesFromObject(t *testing.T) {
	testCases := []struct {
		name          string
		namespace     *corev1.Namespace
		key           string
		wantRuleNames []string
	}{
		{
			name: "should return ruleNames when annotation exists",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-namespace",
					Annotations: map[string]string{
						model.RulesKey: "rule2,rule1,rule3",
					},
				},
			},
			key:           model.RulesKey,
			wantRuleNames: []string{"rule2", "rule1", "rule3"},
		},
		{
			name: "should return ruleNames when annotation exists",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-namespace",
					Annotations: map[string]string{
						model.RulesKey: "rule2, rule1, rule3",
					},
				},
			},
			key:           model.RulesKey,
			wantRuleNames: []string{"rule2", "rule1", "rule3"},
		},
		{
			name: "should return empty slice when annotations is nil",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "example-namespace",
					Annotations: nil,
				},
			},
			key:           "nonExistentKey",
			wantRuleNames: []string{},
		},
		{
			name: "should return empty slice when annotation key does not exist",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "example-namespace",
					Annotations: map[string]string{},
				},
			},
			key:           "nonExistentKey",
			wantRuleNames: []string{},
		},
		{
			name: "should return empty slice when annotation value is empty",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-namespace",
					Annotations: map[string]string{
						model.RulesKey: "",
					},
				},
			},
			key:           model.RulesKey,
			wantRuleNames: []string{},
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			ruleNames := getRuleNamesFromObject(tc.namespace, tc.key)
			assert.Equal(t, tc.wantRuleNames, ruleNames)
		})
	}
}

func TestMergeRuleNames(t *testing.T) {
	tests := []struct {
		name          string
		ruleNames1    []string
		ruleNames2    []string
		wantRuleNames []string
	}{
		{
			name:          "Both slices empty",
			ruleNames1:    []string{},
			ruleNames2:    []string{},
			wantRuleNames: []string{},
		},
		{
			name:          "First slice empty, second slice with elements",
			ruleNames1:    []string{},
			ruleNames2:    []string{"rule1", "rule2"},
			wantRuleNames: []string{"rule1", "rule2"},
		},
		{
			name:          "Second slice empty, first slice with elements",
			ruleNames1:    []string{"rule1", "rule2"},
			ruleNames2:    []string{},
			wantRuleNames: []string{"rule1", "rule2"},
		},
		{
			name:          "No duplicate ruleNames",
			ruleNames1:    []string{"rule1", "rule3"},
			ruleNames2:    []string{"rule2", "rule4"},
			wantRuleNames: []string{"rule1", "rule3", "rule2", "rule4"},
		},
		{
			name:          "Some duplicate ruleNames",
			ruleNames1:    []string{"rule1", "rule3"},
			ruleNames2:    []string{"rule3", "rule4"},
			wantRuleNames: []string{"rule1", "rule3", "rule4"},
		},
		{
			name:          "All ruleNames duplicated",
			ruleNames1:    []string{"rule1", "rule2"},
			ruleNames2:    []string{"rule1", "rule2"},
			wantRuleNames: []string{"rule1", "rule2"},
		},
		{
			name:          "Mixed duplicates and unique ruleNames",
			ruleNames1:    []string{"rule1", "rule3", "rule5"},
			ruleNames2:    []string{"rule2", "rule3", "rule6"},
			wantRuleNames: []string{"rule1", "rule3", "rule5", "rule2", "rule6"},
		},
	}

	for i, tt := range tests {
		t.Run(testcase.Name(i, tt.name), func(t *testing.T) {
			ruleNames := mergeRuleNames(tt.ruleNames1, tt.ruleNames2)
			assert.Equal(t, tt.wantRuleNames, ruleNames)
		})
	}
}

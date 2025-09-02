package ingresscontroller

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kuoss/ingress-annotator/pkg/matcher"
	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/testutil/fakeclient"
	"github.com/kuoss/ingress-annotator/pkg/testutil/mocks"
)

func TestSetupWithManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	store := mocks.NewMockIRulesStore(mockCtrl)
	matcher := matcher.New(store)

	client := fakeclient.NewClient(nil)
	reconciler := &IngressReconciler{
		Client:  client,
		Matcher: matcher,
	}

	err := reconciler.SetupWithManager(fakeclient.NewManager())
	assert.NoError(t, err)
}

func TestReconcile_ok(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		name               string
		clientOpts         *fakeclient.ClientOpts
		ingressAnnotations map[string]string
		deletionTimestamp  *metav1.Time
		finalizers         []string
		wantResult         ctrl.Result
		wantAnnotations    map[string]string
	}{
		{
			name:              "HandleDeletion_WhenIngressHasDeletionTimestampAndFinalizer",
			deletionTimestamp: &metav1.Time{Time: time.Now()},
			finalizers:        []string{"test-finalizer"},

			wantResult: ctrl.Result{},
		},
		{
			name:       "AddDefaultAnnotations_WhenIngressExistsWithoutAnnotations",
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
		},
		{
			name: "ReconcileAnnotations_WhenReconcileAnnotationIsTrue",
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/reconcile": "true",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
		},
		{
			name:               "AddNewAnnotations_WhenValidIngressHasNoMatchingRule",
			ingressAnnotations: map[string]string{},
			wantResult:         ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
		},
		{
			name: "RetainExistingAnnotations_WhenValidIngressHasPreExistingAnnotations",
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
		},
		{
			name: "ReturnEarly_WhenNoAnnotationChangesAreDetected",
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "old-value",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"new-key": "new-value",
			},
		},
		{
			name: "ReconcileAnnotations_WithExtraAnnotationsAndReconcileTrue",
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/reconcile": "true",
				"example-key": "example-value",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"example-key": "example-value",
				"new-key":     "new-value",
			},
		},
		{
			name: "RetainExistingAnnotations_WhenIngressHasExampleAnnotation",
			ingressAnnotations: map[string]string{
				"example-key": "example-value",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"example-key": "example-value",
				"new-key":     "new-value",
			},
		},
		{
			name: "ResetInvalidManagedAnnotations_WhenInvalidJSONIsPresent",
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "invalid-json",
				"example-key": "example-value",
			},
			wantResult: ctrl.Result{},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"example-key": "example-value",
				"new-key":     "new-value",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()
			nn := types.NamespacedName{Namespace: "default", Name: "my-ingress"}

			namespace := &corev1.Namespace{ObjectMeta: ctrl.ObjectMeta{Name: "default"}}
			ingress := &networkingv1.Ingress{
				ObjectMeta: ctrl.ObjectMeta{
					Namespace:         "default",
					Name:              "my-ingress",
					Annotations:       tc.ingressAnnotations,
					DeletionTimestamp: tc.deletionTimestamp,
					Finalizers:        tc.finalizers,
				},
			}
			client := fakeclient.NewClient(tc.clientOpts, namespace, ingress)

			// Mock the rules store
			rules := []model.Rule{{
				Selector:    model.Selector{Include: "*"},
				Description: "new-key-value",
				Annotations: model.Annotations{"new-key": "new-value"},
			}}
			store := mocks.NewMockIRulesStore(mockCtrl)
			store.EXPECT().GetRules().Return(rules).AnyTimes()

			reconciler := &IngressReconciler{
				Client:  client,
				Matcher: matcher.New(store),
			}

			// Run the Reconcile method
			got, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: nn})

			assert.Equal(t, tc.wantResult, got)
			assert.NoError(t, err)

			updatedIngress := &networkingv1.Ingress{}
			err = client.Get(ctx, nn, updatedIngress)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantAnnotations, updatedIngress.Annotations)
		})
	}
}

func TestReconcile_error(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		name               string
		clientOpts         *fakeclient.ClientOpts
		requestNN          *types.NamespacedName
		ingressAnnotations map[string]string
		deletionTimestamp  *metav1.Time
		finalizers         []string
		wantResult         ctrl.Result
		wantAnnotations    map[string]string
		wantError          string
		wantGetError       string
	}{
		{
			name:         "IngressDoesNotExist_ShouldReturnNotFoundError",
			requestNN:    &types.NamespacedName{Namespace: "default", Name: "xxx"},
			wantResult:   ctrl.Result{},
			wantGetError: `ingresses.networking.k8s.io "xxx" not found`,
		},
		{
			name:       "ReconcileAnnotationTrueWithUpdateError_ShouldReturnError",
			clientOpts: &fakeclient.ClientOpts{UpdateError: true},
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/reconcile": "true",
			},
			wantResult: ctrl.Result{RequeueAfter: 30 * time.Second},
			wantAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/reconcile": "true",
			},
			wantError: "mocked UpdateError",
		},
		{
			name:       "ClientGetError_ShouldReturnError",
			clientOpts: &fakeclient.ClientOpts{GetError: "*"},
			ingressAnnotations: map[string]string{
				"example-key": "example-value",
			},
			wantResult: ctrl.Result{},
			wantError:  "mocked GetError",
		},
		{
			name:       "ClientGetErrorWithNamespace_ShouldReturnNamespaceError",
			clientOpts: &fakeclient.ClientOpts{GetError: "Namespace"},
			ingressAnnotations: map[string]string{
				"example-key": "example-value",
			},
			wantResult: ctrl.Result{},
			wantError:  "mocked GetError Namespace",
		},
		{
			name:         "RulesProvidedButIngressNotFound_ShouldReturnNotFoundError",
			clientOpts:   &fakeclient.ClientOpts{GetNotFoundError: true},
			wantResult:   ctrl.Result{},
			wantGetError: "mocked GetNotFoundError: Resource \"my-ingress\" not found",
		},
		{
			name:       "ClientUpdateError_ShouldRequeueAfterError",
			clientOpts: &fakeclient.ClientOpts{UpdateError: true},
			ingressAnnotations: map[string]string{
				"annotator.ingress.kubernetes.io/managed-annotations": "{\"new-key\":\"new-value\"}\n",
				"annotator.ingress.kubernetes.io/rules":               "rule1",
			},
			wantResult: ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError:  "mocked UpdateError",
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			ctx := context.Background()
			nn := types.NamespacedName{Namespace: "default", Name: "my-ingress"}
			if tc.requestNN != nil {
				nn = *tc.requestNN
			}

			namespace := &corev1.Namespace{ObjectMeta: ctrl.ObjectMeta{Name: "default"}}
			ingress := &networkingv1.Ingress{
				ObjectMeta: ctrl.ObjectMeta{
					Namespace:         "default",
					Name:              "my-ingress",
					Annotations:       tc.ingressAnnotations,
					DeletionTimestamp: tc.deletionTimestamp,
					Finalizers:        tc.finalizers,
				},
			}
			client := fakeclient.NewClient(tc.clientOpts, namespace, ingress)

			// Mock the rules store
			rules := []model.Rule{{
				Selector:    model.Selector{Include: "*"},
				Description: "rule1",
				Annotations: model.Annotations{"new-key": "new-value"},
			}}
			store := mocks.NewMockIRulesStore(mockCtrl)
			store.EXPECT().GetRules().Return(rules).AnyTimes()

			reconciler := &IngressReconciler{
				Client:  client,
				Matcher: matcher.New(store),
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
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantAnnotations, updatedIngress.Annotations)
		})
	}
}

func TestGetToBeAnnotations(t *testing.T) {
	ctx := context.TODO()

	// Mock inputs
	namespace := &corev1.Namespace{ObjectMeta: ctrl.ObjectMeta{Name: "default"}}
	ingress := &networkingv1.Ingress{
		ObjectMeta: ctrl.ObjectMeta{
			Namespace: "default",
			Name:      "my-ingress",
			Annotations: model.Annotations{
				"key1":                      "value1",
				model.ReconcileKey:          "true",
				model.ManagedAnnotationsKey: `{"managed-key": "managed-value"}`,
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	rules := []model.Rule{{
		Selector:    model.Selector{Include: "*"},
		Description: "rule1",
		Annotations: model.Annotations{"new-key": "new-value"},
	}}
	store := mocks.NewMockIRulesStore(mockCtrl)
	store.EXPECT().GetRules().Return(rules).AnyTimes()

	client := fakeclient.NewClient(nil, namespace, ingress)
	reconciler := &IngressReconciler{
		Client:  client,
		Matcher: matcher.New(store),
	}
	// Execute
	scope := &ingressScope{
		logger:    logr.Logger{},
		namespace: namespace,
		ingress:   ingress,
	}
	result := reconciler.GetToBeAnnotations(ctx, scope)

	// Expected annotations
	expected := model.Annotations{
		"key1":                      "value1",
		"new-key":                   "new-value",
		model.ManagedAnnotationsKey: "{\"new-key\":\"new-value\"}\n",
	}

	// Validate
	assert.Equal(t, expected, result)
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

	for i, tt := range tests {
		t.Run(testcase.Name(i, tt.name), func(t *testing.T) {
			dst := copyAnnotations(tt.input)
			assert.Equal(t, tt.expectedOutput, dst)

			if tt.modifyCopy {
				dst["key1"] = "modifiedValue"
				assert.Equal(t, "value1", tt.input["key1"], "Original map should not be modified")
			}
		})
	}
}

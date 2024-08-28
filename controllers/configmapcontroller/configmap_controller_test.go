package configmapcontroller

import (
	"context"
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
			name:       "Requeue on ConfigMap Get error",
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			cm:         createConfigMap("default", "ingress-annotator", ""),
			newCM:      createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:         types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:       ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError:  "failed to get ConfigMap: mocked GetError",
		},
		{
			name:       "Requeue when ConfigMap not found",
			clientOpts: &fakeclient.ClientOpts{GetNotFoundError: true},
			cm:         createConfigMap("default", "ingress-annotator", ""),
			newCM:      createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:         types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:       ctrl.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name:       "Error during Ingress list should result in requeue",
			clientOpts: &fakeclient.ClientOpts{ListError: true},
			cm:         createConfigMap("default", "ingress-annotator", ""),
			newCM:      createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:         types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN:  types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:       ctrl.Result{},
			wantError:  "failed to annotateAllIngresses: failed to list ingresses: mocked ListError",
		},
		{
			name:      "Unmarshal error on invalid ConfigMap data",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			newCM:     createConfigMap("default", "ingress-annotator", "invalid rules"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{RequeueAfter: 30 * time.Second},
			wantError: "failed to update rules in rules store: failed to extract rules from configMap: failed to unmarshal rules: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rules",
		},
		{
			name:      "No requeue when ConfigMap has no changes",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			newCM:     createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{},
		},
		{
			name:      "Process valid ConfigMap without errors or requeue",
			cm:        createConfigMap("default", "ingress-annotator", "rule1:\n  key1: value1"),
			nn:        types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			requestNN: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
			want:      ctrl.Result{},
		},
		{
			name:      "No errors when request name differs from ConfigMap name",
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

func TestConfigMapReconciler_annotateAllIngresses(t *testing.T) {
	ingress1 := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ingress1", Namespace: "default"}}
	ingress2 := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ingress2", Namespace: "default"}}

	testCases := []struct {
		clientOpts *fakeclient.ClientOpts
		wantError  string
	}{
		{
			clientOpts: nil,
			wantError:  "",
		},
		{
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			wantError:  "failed to annotateIngress: failed to get ingress default/ingress1: mocked GetError",
		},
		{
			clientOpts: &fakeclient.ClientOpts{UpdateError: true},
			wantError:  "failed to annotateIngress: failed to update ingress default/ingress1: mocked UpdateError",
		},
		{
			clientOpts: &fakeclient.ClientOpts{UpdateConflictError: true},
			wantError:  "failed to annotateIngress: mocked UpdateConflictError: Operation cannot be fulfilled on ingresses.networking.k8s.io \"ingress1\": the object has been modified; please apply your changes to the latest version and try again",
		},
	}
	for i, tc := range testCases {
		t.Run(testcase.Name(i), func(t *testing.T) {
			client := fakeclient.NewClient(tc.clientOpts, ingress1, ingress2)
			reconciler := &ConfigMapReconciler{
				Client: client,
			}
			err := reconciler.annotateAllIngresses(context.TODO())
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}

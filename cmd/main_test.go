package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"testing"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kuoss/ingress-annotator/pkg/testutil/fakeclient"
	"github.com/kuoss/ingress-annotator/pkg/testutil/mocks"
)

func TestGetManagerOptions(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	metricsAddr := fs.String("metrics-bind-address", "0", "")
	probeAddr := fs.String("health-probe-bind-address", ":8081", "")
	enableLeaderElection := fs.Bool("leader-elect", false, "")
	secureMetrics := fs.Bool("metrics-secure", true, "")
	enableHTTP2 := fs.Bool("enable-http2", false, "")
	err := fs.Parse([]string{})
	assert.NoError(t, err)

	opts := getManagerOptions()
	assert.Equal(t, *metricsAddr, opts.Metrics.BindAddress, "Expected metrics bind address to match")
	assert.Equal(t, *probeAddr, opts.HealthProbeBindAddress, "Expected health probe bind address to match")
	assert.Equal(t, *enableLeaderElection, opts.LeaderElection, "Expected leader election setting to match")
	assert.Equal(t, *secureMetrics, opts.Metrics.SecureServing, "Expected secure metrics setting to match")

	webhookServer, ok := opts.WebhookServer.(*webhook.DefaultServer)
	if !ok || webhookServer == nil {
		t.Fatal("Expected a valid webhook.Server instance")
	}

	// If enableHTTP2 is false, check the TLS options indirectly
	if !*enableHTTP2 {
		tlsConfig := &tls.Config{}
		for _, tlsOpt := range webhookServer.Options.TLSOpts {
			tlsOpt(tlsConfig)
		}
		assert.Equal(t, []string{"http/1.1"}, tlsConfig.NextProtos, "Expected HTTP/2 to be disabled")
	}

	// Check the default leader election ID
	assert.Equal(t, "annotator.ingress.kubernetes.io", opts.LeaderElectionID, "Expected leader election ID to match")
}

type managerOpts struct {
	clientOpts         *fakeclient.ClientOpts
	AddHealthzCheckErr error
	AddReadyzCheckErr  error
	StartErr           error
}

func setupMockManager(mockCtrl *gomock.Controller, opts *managerOpts, objs ...client.Object) *mocks.MockManager {
	if opts == nil {
		opts = &managerOpts{}
	}
	mockManager := mocks.NewMockManager(mockCtrl)
	mockCache := mocks.NewMockCache(mockCtrl)

	scheme := fakeclient.NewScheme()
	fakeClient := fakeclient.NewClient(opts.clientOpts, objs...)

	mockManager.EXPECT().GetCache().Return(mockCache).AnyTimes()
	mockManager.EXPECT().GetClient().Return(fakeClient).AnyTimes()
	mockManager.EXPECT().GetScheme().Return(scheme).AnyTimes()
	mockManager.EXPECT().GetControllerOptions().Return(config.Controller{}).AnyTimes()
	mockManager.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	mockManager.EXPECT().AddHealthzCheck(gomock.Any(), gomock.Any()).Return(opts.AddHealthzCheckErr).AnyTimes()
	mockManager.EXPECT().AddReadyzCheck(gomock.Any(), gomock.Any()).Return(opts.AddReadyzCheckErr).AnyTimes()
	mockManager.EXPECT().GetLogger().Return(zap.New(zap.WriteTo(nil))).AnyTimes()
	mockManager.EXPECT().GetAPIReader().Return(fakeClient).AnyTimes()
	mockManager.EXPECT().Start(gomock.Any()).Return(opts.StartErr).AnyTimes()

	return mockManager
}

func TestRun(t *testing.T) {
	testCases := []struct {
		name              string
		namespace         string
		managerOpts       *managerOpts
		cm                *corev1.ConfigMap
		setupManagerError func(mgr *mocks.MockManager)
		wantError         string
	}{
		{
			name:      "no error with empty rules",
			namespace: "test-namespace",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace", Name: "ingress-annotator"},
				Data:       map[string]string{"rules": ""},
			},
		},
		{
			name:      "POD_NAMESPACE environment variable is empty",
			namespace: "",
			wantError: "POD_NAMESPACE environment variable is not set or is empty",
		},
		{
			name:        "Error fetching ConfigMap due to mock GetError",
			namespace:   "test-namespace",
			managerOpts: &managerOpts{clientOpts: &fakeclient.ClientOpts{GetError: true}},
			wantError:   "failed to fetch ConfigMap: mocked Get error",
		},
		{
			name:      "Error fetching ConfigMap - ConfigMap not found",
			namespace: "test-namespace",
			cm:        &corev1.ConfigMap{},
			wantError: `failed to fetch ConfigMap: configmaps "ingress-annotator" not found`,
		},
		{
			name:      "Error setting up health check",
			namespace: "test-namespace",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace", Name: "ingress-annotator"},
				Data:       map[string]string{"rules": "invalid rules"},
			},
			wantError: "unable to start rules store: failed to initialize RulesStore: failed to extract rules from configMap: failed to unmarshal rules: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rules",
		},
		{
			name:      "Error setting up ready check",
			namespace: "test-namespace",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace", Name: "ingress-annotator"},
				Data:       map[string]string{"rules": ""},
			},
			managerOpts: &managerOpts{
				AddHealthzCheckErr: errors.New("mocked AddHealthzCheckErr"),
			},
			wantError: "unable to set up health check: mocked AddHealthzCheckErr",
		},
		{
			name:      "Error starting manager",
			namespace: "test-namespace",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace", Name: "ingress-annotator"},
				Data:       map[string]string{"rules": ""},
			},
			managerOpts: &managerOpts{
				AddReadyzCheckErr: errors.New("mocked AddReadyzCheckErr"),
			},
			wantError: "unable to set up ready check: mocked AddReadyzCheckErr",
		},
		{
			name:      "no error",
			namespace: "test-namespace",
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-namespace", Name: "ingress-annotator"},
				Data:       map[string]string{"rules": ""},
			},
			managerOpts: &managerOpts{
				StartErr: errors.New("mocked StartErr"),
			},
			wantError: "problem running manager: mocked StartErr",
		},
	}

	for i, tc := range testCases {
		t.Run(testcase.Name(i, tc.name), func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			t.Setenv("POD_NAMESPACE", tc.namespace)
			mgr := setupMockManager(mockCtrl, tc.managerOpts, tc.cm)
			if tc.setupManagerError != nil {
				tc.setupManagerError(mgr)
			}
			err := run(mgr, context.TODO())
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}

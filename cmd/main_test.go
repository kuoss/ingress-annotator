package main

import (
	"crypto/tls"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kuoss/ingress-annotator/cmd/mocks"
	"github.com/kuoss/ingress-annotator/controller/fakeclient"
	"github.com/kuoss/ingress-annotator/controller/model"
	"github.com/kuoss/ingress-annotator/controller/rulesstore/mockrulesstore"
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

func setupMockManager(mockCtrl *gomock.Controller) (*mocks.MockManager, *mocks.MockCache) {
	mockManager := mocks.NewMockManager(mockCtrl)
	mockCache := mocks.NewMockCache(mockCtrl)

	scheme := fakeclient.NewScheme()

	fakeClient := fakeclient.NewClient(nil, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "ingress-annotator",
		},
	})

	mockManager.EXPECT().GetCache().Return(mockCache).AnyTimes()
	mockManager.EXPECT().GetClient().Return(fakeClient).AnyTimes()
	mockManager.EXPECT().GetScheme().Return(scheme).AnyTimes()
	mockManager.EXPECT().GetControllerOptions().Return(config.Controller{}).AnyTimes()
	mockManager.EXPECT().AddHealthzCheck(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockManager.EXPECT().AddReadyzCheck(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockManager.EXPECT().GetLogger().Return(zap.New(zap.WriteTo(nil))).AnyTimes()

	return mockManager, mockCache
}

func TestRun_PODNamespaceNotSet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockManager, mockCache := setupMockManager(mockCtrl)

	mockCache.EXPECT().WaitForCacheSync(gomock.Any()).Return(true).Times(1)

	t.Setenv("POD_NAMESPACE", "")
	err := run(mockManager)
	assert.Error(t, err)
	assert.Equal(t, "POD_NAMESPACE environment variable is not set or is empty", err.Error())
}

func TestRun_WaitForCacheSyncReturnsFalse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockManager, mockCache := setupMockManager(mockCtrl)

	mockCache.EXPECT().WaitForCacheSync(gomock.Any()).Return(false).Times(1)

	t.Setenv("POD_NAMESPACE", "test-namespace")
	err := run(mockManager)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to wait for cache sync")
}

func TestRun_SuccessfulRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockManager, mockCache := setupMockManager(mockCtrl)

	mockCache.EXPECT().WaitForCacheSync(gomock.Any()).Return(true).Times(1)
	mockManager.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()

	mockRulesStore := new(mockrulesstore.RulesStore)
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

	mockManager.EXPECT().Start(gomock.Any()).Return(nil).Times(1)

	t.Setenv("POD_NAMESPACE", "test-namespace")
	err := run(mockManager)
	assert.NoError(t, err)
}

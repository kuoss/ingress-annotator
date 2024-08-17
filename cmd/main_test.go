package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/kuoss/ingress-annotator/cmd/mocks"
	"github.com/kuoss/ingress-annotator/controller/fakeclient"
)

func TestRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockManager := mocks.NewMockManager(mockCtrl)
	mockCache := mocks.NewMockCache(mockCtrl)

	// Create a fake client with the necessary ConfigMap
	fakeClient := fakeclient.NewClient(nil, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "ingress-annotator",
		},
	})

	// Mock expectations
	mockManager.EXPECT().GetCache().Return(mockCache).Times(1)
	mockCache.EXPECT().WaitForCacheSync(gomock.Any()).Return(true).Times(1)
	mockManager.EXPECT().GetClient().Return(fakeClient).Times(2)
	mockManager.EXPECT().GetScheme().Return(nil).Times(2)
	mockManager.EXPECT().AddHealthzCheck(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockManager.EXPECT().AddReadyzCheck(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockManager.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
	mockManager.EXPECT().GetControllerOptions().Return(controller.Options{}).Times(1)

	// Setting an environment variable for the namespace
	t.Setenv("POD_NAMESPACE", "test-namespace")

	// Run the function and check for errors
	err := run(mockManager)
	assert.NoError(t, err)
}

package rulesstore

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNew(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "default")

	fakeClient := fake.NewFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"rule1": "annotation1: value1\nannotation2: value2",
		},
	})

	rulesStore, err := New(fakeClient)
	assert.NoError(t, err)
	assert.Equal(t, "default", rulesStore.ConfigMapNamespace())
	assert.Equal(t, configMapName, rulesStore.ConfigMapName())

	data := rulesStore.GetData()
	assert.Equal(t, "annotation1: value1\nannotation2: value2", data.ConfigMap.Data["rule1"])
	assert.Equal(t, "value1", data.Rules["rule1"]["annotation1"])
	assert.Equal(t, "value2", data.Rules["rule1"]["annotation2"])
}

func TestNew_ErrorMissingPodNamespace(t *testing.T) {
	err := os.Unsetenv("POD_NAMESPACE")
	assert.NoError(t, err)

	fakeClient := fake.NewFakeClient()

	_, err = New(fakeClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "POD_NAMESPACE environment variable is not set or is empty")
}

func TestNew_ErrorUpdateData(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "default")

	fakeClient := fake.NewFakeClient()

	_, err := New(fakeClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update data error")
}

func TestUpdateData(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "default")

	fakeClient := fake.NewFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"rule1": "annotation1: value1\nannotation2: value2",
		},
	})

	rulesStore, err := New(fakeClient)
	assert.NoError(t, err)

	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"rule2": "annotation3: value3\nannotation4: value4",
		},
	}
	err = fakeClient.Update(context.Background(), newConfigMap)
	assert.NoError(t, err)

	err = rulesStore.UpdateData()
	assert.NoError(t, err)

	data := rulesStore.GetData()
	assert.Equal(t, "annotation3: value3\nannotation4: value4", data.ConfigMap.Data["rule2"])
	assert.Equal(t, "value3", data.Rules["rule2"]["annotation3"])
	assert.Equal(t, "value4", data.Rules["rule2"]["annotation4"])
}

func TestUpdateData_ErrorGetConfigMap(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "default")

	fakeClient := fake.NewFakeClient()

	rulesStore := &RulesStore{
		Client:    fakeClient,
		Meta:      types.NamespacedName{Namespace: "default", Name: configMapName},
		Data:      Data{},
		DataMutex: &sync.Mutex{},
	}

	err := rulesStore.UpdateData()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get ConfigMap")
}

func TestUpdateData_ErrorInvalidConfigMap(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "default")

	// Create a fake client with a ConfigMap that contains invalid data
	fakeClient := fake.NewFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: "default",
		},
		Data: map[string]string{
			"rule1": "invalid_annotation_format",
		},
	})

	rulesStore := &RulesStore{
		Client:    fakeClient,
		Meta:      types.NamespacedName{Namespace: "default", Name: configMapName},
		Data:      Data{},
		DataMutex: &sync.Mutex{},
	}

	err := rulesStore.UpdateData()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data in ConfigMap key rule1")
}

func TestGetAnnotationsFromValue(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		want      Annotations
		wantError bool
	}{
		{
			name:  "Simple annotations",
			input: "annotation1: value1\nannotation2: value2",
			want: Annotations{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			wantError: false,
		},
		{
			name: "Complex annotations without quotes",
			input: `nginx.ingress.kubernetes.io/auth-signin: https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri
nginx.ingress.kubernetes.io/auth-url: https://oauth2-proxy.example.com//oauth2/auth
nginx.ingress.kubernetes.io/whitelist-source-range: 1.1.1.1,2.2.2.2
`,
			want: Annotations{
				"nginx.ingress.kubernetes.io/auth-signin":            "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri",
				"nginx.ingress.kubernetes.io/auth-url":               "https://oauth2-proxy.example.com//oauth2/auth",
				"nginx.ingress.kubernetes.io/whitelist-source-range": "1.1.1.1,2.2.2.2",
			},
			wantError: false,
		},
		{
			name: "Complex annotations with quotes",
			input: `nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com//oauth2/auth"
nginx.ingress.kubernetes.io/whitelist-source-range: "1.1.1.1,2.2.2.2"
`,
			want: Annotations{
				"nginx.ingress.kubernetes.io/auth-signin":            "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri",
				"nginx.ingress.kubernetes.io/auth-url":               "https://oauth2-proxy.example.com//oauth2/auth",
				"nginx.ingress.kubernetes.io/whitelist-source-range": "1.1.1.1,2.2.2.2",
			},
			wantError: false,
		},
		{
			name:  "Annotations with extra newlines and spaces",
			input: "annotation1: value1\n\nannotation2:     value2",
			want: Annotations{
				"annotation1": "value1",
				"annotation2": "value2",
			},
			wantError: false,
		},
		{
			name:      "Invalid annotations with incorrect indentation",
			input:     "annotation1: value1\n\n    annotation2:    value2",
			want:      nil,
			wantError: true,
		},
		{
			name:      "Invalid single annotation",
			input:     "invalid_annotation",
			want:      nil,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			annotations, err := getAnnotationsFromText(tc.input)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, annotations)
		})
	}
}

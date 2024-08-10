package rulesstore

import (
	"sync"
	"testing"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestRulesStore(t *testing.T) {
	mockData := map[string]string{
		"rule1": `
annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
		"rule2": `
annotations:
  key2: value2
namespace: another-namespace`,
	}
	cm := &corev1.ConfigMap{
		Data: mockData,
	}

	store := New()
	err := store.UpdateRules(cm)
	assert.NoError(t, err)

	want := &model.Rules{
		"rule1": {
			Annotations: model.Annotations{"key1": "value1"},
			Namespace:   "test-namespace",
			Ingress:     "test-ingress",
		},
		"rule2": {
			Annotations: model.Annotations{"key2": "value2"},
			Namespace:   "another-namespace",
			Ingress:     "",
		},
	}
	got := store.GetRules()
	assert.Equal(t, want, got)
}

func TestGetRuleValueFromText(t *testing.T) {
	yamlText := `
annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`
	want := &model.Rule{
		Annotations: model.Annotations{"key1": "value1"},
		Namespace:   "test-namespace",
		Ingress:     "test-ingress",
	}

	got, err := getRuleValueFromText(yamlText)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestConcurrency(t *testing.T) {
	store := New()
	store.Rules = &model.Rules{
		"initialRule": {
			Annotations: model.Annotations{"initialKey": "initialValue"},
			Namespace:   "initial-namespace",
			Ingress:     "initial-ingress",
		},
	}

	var wg sync.WaitGroup

	// Test concurrent access to GetRules and UpdateRules
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetRules()
		}()

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mockData := map[string]string{
				"rule": `
annotations:
  key: value
namespace: namespace`,
			}
			cm := &corev1.ConfigMap{
				Data: mockData,
			}
			_ = store.UpdateRules(cm)
		}(i)
	}

	wg.Wait()
}

func TestInvalidData(t *testing.T) {
	store := New()
	invalidData := map[string]string{
		"rule1": `invalid_yaml`,
	}
	cm := &corev1.ConfigMap{
		Data: invalidData,
	}

	err := store.UpdateRules(cm)
	assert.ErrorContains(t, err, "yaml: unmarshal errors:\n  line 1")
}

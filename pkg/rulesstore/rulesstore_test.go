package rulesstore

import (
	"sync"
	"testing"

	"github.com/jmnote/tester/testcase"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/kuoss/ingress-annotator/pkg/model"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cm        *corev1.ConfigMap
		wantRules *model.Rules
		wantError string
	}{
		{
			name:      "Nil ConfigMap",
			cm:        nil,
			wantRules: nil,
			wantError: "failed to initialize RulesStore: failed to extract rules from configMap: configMap is nil",
		},
		{
			name: "Valid ConfigMap",
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rules": `
rule1:
  key1: value1`,
				},
			},
			wantRules: &model.Rules{
				"rule1": model.Annotations{"key1": "value1"},
			},
			wantError: "",
		},
	}

	for i, tt := range tests {
		t.Run(testcase.Name(i, tt.name), func(t *testing.T) {
			store, err := New(tt.cm)

			if tt.wantError != "" {
				assert.Nil(t, store)
				assert.EqualError(t, err, tt.wantError)
			} else {
				assert.NotNil(t, store)
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRules, store.GetRules())
			}
		})
	}
}

func TestGetRules(t *testing.T) {
	wantRules := &model.Rules{
		"rule1": model.Annotations{"key1": "value1"},
	}

	store := &RulesStore{
		Rules:      wantRules,
		rulesMutex: &sync.Mutex{},
	}

	gotRules := store.GetRules()

	assert.Equal(t, wantRules, gotRules)
}

func TestUpdateRules(t *testing.T) {
	tests := []struct {
		name      string
		cm        *corev1.ConfigMap
		wantRules *model.Rules
		wantError string
	}{
		{
			name:      "Nil ConfigMap",
			cm:        nil,
			wantError: "failed to extract rules from configMap: configMap is nil",
		},
		{
			name:      "Empty ConfigMap",
			cm:        &corev1.ConfigMap{},
			wantError: "failed to extract rules from configMap: configMap missing 'rules' key",
		},
		{
			name: "Invalid YAML in ConfigMap",
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rules": `
rule1:
  invalid_data`,
				},
			},
			wantError: "failed to extract rules from configMap: failed to unmarshal rules: yaml: unmarshal errors:\n  line 3: cannot unmarshal !!str `invalid...` into model.Annotations",
		},
		{
			name: "Valid ConfigMap",
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rules": `
rule1:
  key1: value1`,
				},
			},
			wantRules: &model.Rules{
				"rule1": model.Annotations{"key1": "value1"},
			},
		},
	}

	for i, tt := range tests {
		t.Run(testcase.Name(i, tt.name), func(t *testing.T) {
			store := &RulesStore{
				rulesMutex: &sync.Mutex{},
			}
			err := store.UpdateRules(tt.cm)

			if tt.wantError != "" {
				assert.EqualError(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantRules != nil {
				assert.Equal(t, tt.wantRules, store.GetRules())
			}
		})
	}
}

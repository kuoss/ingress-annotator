package rulesstore

import (
	"sync"
	"testing"

	"github.com/jmnote/tester/testcase"
	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cm        *corev1.ConfigMap
		wantRules []model.Rule
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
- description: rule1
  annotations:
    key1: value1
`,
				},
			},
			wantRules: []model.Rule{
				{
					Description: "rule1",
					Annotations: model.Annotations{"key1": "value1"},
				},
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
	wantRules := []model.Rule{{
		Description: "rule1",
		Annotations: model.Annotations{"key1": "value1"},
	}}
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
		wantRules []model.Rule
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
- description: rule1
  annotations:
    invalid_data`},
			},
			wantError: "failed to extract rules from configMap: failed to unmarshal rules: yaml: unmarshal errors:\n  line 4: cannot unmarshal !!str `invalid...` into model.Annotations",
		},
		{
			name: "Valid ConfigMap",
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rules": `
- description: rule1
  annotations:
    key1: value1`},
			},
			wantRules: []model.Rule{{
				Description: "rule1",
				Annotations: model.Annotations{"key1": "value1"},
			}},
		},
		{
			name: "Valid ConfigMap",
			cm: &corev1.ConfigMap{
				Data: map[string]string{"rules": "- description: rule1\n  annotations:\n    key1: value1"},
			},
			wantRules: []model.Rule{{
				Description: "rule1",
				Annotations: model.Annotations{"key1": "value1"},
			}},
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

func TestGetRulesFromConfigMap(t *testing.T) {
	t.Run("Nil ConfigMap", func(t *testing.T) {
		rules, err := getRulesFromConfigMap(nil)
		assert.Nil(t, rules)
		assert.EqualError(t, err, "configMap is nil")
	})

	t.Run("Missing rules key", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Data: map[string]string{
				"other_key": "value",
			},
		}
		rules, err := getRulesFromConfigMap(cm)
		assert.Nil(t, rules)
		assert.EqualError(t, err, "configMap missing 'rules' key")
	})

	t.Run("Invalid YAML in rules", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Data: map[string]string{
				"rules": "invalid_yaml: [",
			},
		}
		rules, err := getRulesFromConfigMap(cm)
		assert.Nil(t, rules)
		assert.ErrorContains(t, err, "failed to unmarshal rules")
	})

	t.Run("Valid rules", func(t *testing.T) {
		validRules := []model.Rule{
			{
				Description: "rule1",
				Selector: model.Selector{
					Include: "app1",
					Exclude: "app2",
				},
				Annotations: model.Annotations{
					"key1": "value1",
				},
			},
		}
		rulesYaml, err := yaml.Marshal(validRules)
		assert.NoError(t, err)

		cm := &corev1.ConfigMap{
			Data: map[string]string{
				"rules": string(rulesYaml),
			},
		}
		rules, err := getRulesFromConfigMap(cm)
		assert.NoError(t, err)
		assert.Equal(t, validRules, rules)
	})
}

package rulesstore

import (
	"sync"
	"testing"

	"github.com/jmnote/tester"
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
		go func(_ int) {
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

func TestUpdateRules(t *testing.T) {
	tests := []struct {
		name          string
		configMapData map[string]string
		expectError   bool
	}{
		{
			name: "Valid rule",
			configMapData: map[string]string{
				"rule1": "namespace: test-namespace\ningress: test-ingress",
			},
			expectError: false,
		},
		{
			name: "Invalid namespace pattern",
			configMapData: map[string]string{
				"rule1": "namespace: invalid_namespace!\ningress: test-ingress",
			},
			expectError: true,
		},
		{
			name: "Invalid ingress pattern",
			configMapData: map[string]string{
				"rule1": "namespace: test-namespace\ningress: invalid_ingress!",
			},
			expectError: true,
		},
		{
			name: "Invalid YAML",
			configMapData: map[string]string{
				"rule1": "namespace: test-namespace\ningress",
			},
			expectError: true,
		},
		{
			name:          "Empty ConfigMap",
			configMapData: map[string]string{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := New()
			cm := &corev1.ConfigMap{
				Data: tt.configMapData,
			}

			err := rs.UpdateRules(cm)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestValidateRule(t *testing.T) {
	testCases := []struct {
		name    string
		rule    model.Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid Rule with Namespace and Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "ingress-1",
			},
			wantErr: false,
		},
		{
			name: "Valid Rule with Namespace and empty Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "",
			},
			wantErr: false,
		},
		{
			name: "Invalid Namespace pattern",
			rule: model.Rule{
				Namespace: "Invalid_Namespace",
				Ingress:   "ingress-1",
			},
			wantErr: true,
			errMsg:  "invalid namespace pattern: Invalid_Namespace",
		},
		{
			name: "Invalid Ingress pattern",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "Invalid,Ingress",
			},
			wantErr: true,
			errMsg:  "invalid ingress pattern: Invalid,Ingress",
		},
		{
			name: "Valid Namespace with wildcard and negation",
			rule: model.Rule{
				Namespace: "namespace-*",
				Ingress:   "!ingress-*",
			},
			wantErr: false,
		},
		{
			name: "Invalid pattern with special characters",
			rule: model.Rule{
				Namespace: "namespace-!",
				Ingress:   "ingress-1",
			},
			wantErr: true,
			errMsg:  "invalid namespace pattern: namespace-!",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			err := validateRule(&tc.rule)
			if tc.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestValidate(t *testing.T) {
	testCases := []struct {
		pattern string
		want    bool
	}{
		// true
		{"", true},                 // Empty string
		{"abc", true},              // Single string
		{"abc,def", true},          // Comma-separated strings
		{"!abc", true},             // Pattern with an exclamation mark
		{"!abc,def", true},         // Pattern with exclamation mark and comma-separated strings
		{"abc-def", true},          // String with a hyphen
		{"abc*", true},             // String with an asterisk
		{"abc,def,ghi", true},      // Multiple comma-separated strings
		{"!abc,def-ghi", true},     // Pattern with exclamation mark and hyphen
		{"abc,def-ghi,jkl*", true}, // Combination of comma, hyphen, and asterisk
		// false
		{"abc,", false},          // Invalid pattern ending with a comma
		{"abc,def!", false},      // Invalid pattern with an exclamation mark in the wrong place
		{"!abc,", false},         // Invalid pattern ending with a comma after exclamation mark
		{"abc,def*", true},       // Pattern with an asterisk
		{"!abc,def*", true},      // Pattern with exclamation mark and asterisk
		{"abc@def", false},       // Invalid pattern with an incorrect special character
		{"abc,def,", false},      // Invalid comma placement
		{"!abc,def,ghi!", false}, // Exclamation mark in the wrong place
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.pattern), func(t *testing.T) {
			got := validate(tc.pattern)
			assert.Equal(t, tc.want, got)
		})
	}
}

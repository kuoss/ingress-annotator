package rulesstore

import (
	"testing"

	"github.com/jmnote/tester"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/kuoss/ingress-annotator/pkg/model"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		cm        *corev1.ConfigMap
		want      *RulesStore
		wantError string
	}{
		{
			cm:        nil,
			wantError: "store.UpdateRules err: configMap is nil",
		},
		{
			cm: &corev1.ConfigMap{},
			want: &RulesStore{
				Rules: &model.Rules{},
			},
		},
		{
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rule1": "",
				},
			},
			want: &RulesStore{
				Rules: &model.Rules{},
			},
		},
		{
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rule1": `annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
				},
			},
			want: &RulesStore{
				Rules: &model.Rules{
					"rule1": model.Rule{
						Annotations: model.Annotations{"key1": "value1"},
						Namespace:   "test-namespace",
						Ingress:     "test-ingress",
					}},
			},
		},
		{
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rule1": `annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
				},
			},
			want: &RulesStore{
				Rules: &model.Rules{
					"rule1": model.Rule{
						Annotations: model.Annotations{"key1": "value1"},
						Namespace:   "test-namespace",
						Ingress:     "test-ingress",
					}},
			},
		},
		{
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rule1": `invalid_data`,
				},
			},
			wantError: "store.UpdateRules err: invalid data in ConfigMap key rule1: failed to unmarshal YAML: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rule",
		},
		{
			cm: &corev1.ConfigMap{
				Data: map[string]string{
					"rule1": `namespace: invalid_namespace!`,
				},
			},
			wantError: "store.UpdateRules err: validateRule err: invalid namespace pattern: invalid_namespace!",
		},
	}
	for i, tc := range testCases {
		t.Run(tester.Name(i, tc), func(t *testing.T) {
			got, err := New(tc.cm)
			if tc.wantError == "" {
				assert.NoError(t, err)
				tc.want.rulesMutex = got.rulesMutex
				assert.Equal(t, tc.want, got)
				assert.Equal(t, tc.want.Rules, got.GetRules())
			} else {
				assert.EqualError(t, err, tc.wantError)
				assert.Equal(t, got, tc.want)
			}
		})
	}
}

func TestValidateRule(t *testing.T) {
	testCases := []struct {
		name      string
		rule      model.Rule
		wantError string
	}{
		{
			name: "Valid Rule with Namespace and Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "ingress-1",
			},
			wantError: "",
		},
		{
			name: "Valid Rule with Namespace and empty Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "",
			},
			wantError: "",
		},
		{
			name: "Invalid Namespace pattern",
			rule: model.Rule{
				Namespace: "Invalid_Namespace",
				Ingress:   "ingress-1",
			},
			wantError: "invalid namespace pattern: Invalid_Namespace",
		},
		{
			name: "Invalid Ingress pattern",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "Invalid,Ingress",
			},
			wantError: "invalid ingress pattern: Invalid,Ingress",
		},
		{
			name: "Valid Namespace with wildcard and negation",
			rule: model.Rule{
				Namespace: "namespace-*",
				Ingress:   "!ingress-*",
			},
			wantError: "",
		},
		{
			name: "Invalid pattern with special characters",
			rule: model.Rule{
				Namespace: "namespace-!",
				Ingress:   "ingress-1",
			},
			wantError: "invalid namespace pattern: namespace-!",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			err := validateRule(&tc.rule)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
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

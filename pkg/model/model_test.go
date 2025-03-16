package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestUnmarshal(t *testing.T) {
	wantRules := []Rule{
		{
			Description: "oauth2-proxy",
			Annotations: Annotations{
				"nginx.ingress.kubernetes.io/auth-signin": "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri",
				"nginx.ingress.kubernetes.io/auth-url":    "https://oauth2-proxy.example.com/oauth2/auth",
			},
		},
		{
			Description: "private",
			Annotations: Annotations{
				"nginx.ingress.kubernetes.io/whitelist-source-range": "192.168.1.0/24,10.0.0.0/16",
			},
		},
	}
	rulesText := `
- description: oauth2-proxy
  annotations:
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com/oauth2/auth"
- description: private
  annotations:
    nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.1.0/24,10.0.0.0/16"
`

	var rules []Rule
	err := yaml.Unmarshal([]byte(rulesText), &rules)
	assert.NoError(t, err)
	assert.Equal(t, wantRules, rules)
}

// TestExtendedRule_ToRule verifies that ToRule correctly converts XRule to Rule
// by merging listAnnotations into annotations without conflicts.
func TestExtendedRule_ToRule(t *testing.T) {
	extendedRuleText := `
- description: multi-value example
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://auth.example.com/check"
  listAnnotations:
    nginx.ingress.kubernetes.io/whitelist-source-range:
      - "192.168.1.0/24"
      - "10.0.0.0/16"
`

	wantRule := Rule{
		Description: "multi-value example",
		Annotations: Annotations{
			"nginx.ingress.kubernetes.io/auth-url":               "https://auth.example.com/check",
			"nginx.ingress.kubernetes.io/whitelist-source-range": "192.168.1.0/24,10.0.0.0/16",
		},
	}

	var xRules []XRule
	err := yaml.Unmarshal([]byte(extendedRuleText), &xRules)
	assert.NoError(t, err)
	assert.Len(t, xRules, 1)

	gotRule, err := xRules[0].ToRule()
	assert.NoError(t, err)             // Ensure no error occurs during conversion
	assert.Equal(t, wantRule, gotRule) // Verify the converted Rule matches the expected result
}

// TestExtendedRule_ToRule_ConflictError ensures that ToRule returns an error
// when there is a conflicting key in both annotations and listAnnotations.
func TestExtendedRule_ToRule_ConflictError(t *testing.T) {
	extendedRuleText := `
- description: conflict example
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://auth.example.com/check"
  listAnnotations:
    nginx.ingress.kubernetes.io/auth-url:
      - "https://duplicate.example.com"
`

	var xRules []XRule
	err := yaml.Unmarshal([]byte(extendedRuleText), &xRules)
	assert.NoError(t, err)
	assert.Len(t, xRules, 1)

	_, err = xRules[0].ToRule()
	assert.Error(t, err)                                     // Expect an error due to key conflict
	assert.Contains(t, err.Error(), "conflicting key found") // Ensure the error message indicates the conflict
}

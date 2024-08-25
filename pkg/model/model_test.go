package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestUnmarshal(t *testing.T) {
	wantRules := Rules{
		"oauth2-proxy": Annotations{
			"nginx.ingress.kubernetes.io/auth-signin": "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri",
			"nginx.ingress.kubernetes.io/auth-url":    "https://oauth2-proxy.example.com/oauth2/auth",
		},
		"private": Annotations{
			"nginx.ingress.kubernetes.io/whitelist-source-range": "192.168.1.0/24,10.0.0.0/16",
		},
	}
	rulesText := `
oauth2-proxy:
  nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.example.com/oauth2/start?rd=https://$host$request_uri"
  nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.example.com/oauth2/auth"
private:
  nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.1.0/24,10.0.0.0/16"
`

	var rules Rules
	err := yaml.Unmarshal([]byte(rulesText), &rules)
	assert.NoError(t, err)
	assert.Equal(t, wantRules, rules)
}

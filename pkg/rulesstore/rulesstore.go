package rulesstore

import (
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	"github.com/kuoss/ingress-annotator/pkg/model"
)

type IRulesStore interface {
	GetRules() *model.Rules
	UpdateRules(cm *corev1.ConfigMap) error
}

type RulesStore struct {
	Rules      *model.Rules
	rulesMutex *sync.Mutex
}

func New() *RulesStore {
	return &RulesStore{
		rulesMutex: &sync.Mutex{},
	}
}

func (s *RulesStore) GetRules() *model.Rules {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	return s.Rules
}

func (s *RulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	newRules := model.Rules{}
	for key, text := range cm.Data {
		value, err := getRuleValueFromText(text)
		if err != nil {
			return fmt.Errorf("invalid data in ConfigMap key %s: %w", key, err)
		}
		newRules[key] = *value
	}

	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	s.Rules = &newRules
	return nil
}

func getRuleValueFromText(text string) (*model.Rule, error) {
	var rule model.Rule
	err := yaml.Unmarshal([]byte(text), &rule)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}
	return &rule, nil
}

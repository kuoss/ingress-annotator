package rulesstore

import (
	"errors"
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

func New(cm *corev1.ConfigMap) (*RulesStore, error) {
	store := &RulesStore{
		rulesMutex: &sync.Mutex{},
	}
	if err := store.UpdateRules(cm); err != nil {
		return nil, fmt.Errorf("failed to initialize RulesStore: %w", err)
	}
	return store, nil
}

func (s *RulesStore) GetRules() *model.Rules {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	return s.Rules
}

func (s *RulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	rules, err := getRulesFromConfigMap(cm)
	if err != nil {
		return fmt.Errorf("failed to extract rules from configMap: %w", err)
	}

	s.updateRules(rules)
	return nil
}

func (s *RulesStore) updateRules(rules model.Rules) {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	s.Rules = &rules
}

func getRulesFromConfigMap(cm *corev1.ConfigMap) (model.Rules, error) {
	if cm == nil {
		return nil, errors.New("configMap is nil")
	}

	rulesText, ok := cm.Data["rules"]
	if !ok {
		return nil, errors.New("configMap missing 'rules' key")
	}

	var rules model.Rules
	if err := yaml.Unmarshal([]byte(rulesText), &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	return rules, nil
}

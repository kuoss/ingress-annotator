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
	GetRules() []model.Rule
	UpdateRules(cm *corev1.ConfigMap) error
}

type RulesStore struct {
	rules      []model.Rule
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

func (s *RulesStore) GetRules() []model.Rule {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	return s.rules
}

func (s *RulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	rules, err := getRulesFromConfigMap(cm)
	if err != nil {
		return fmt.Errorf("failed to extract rules from configMap: %w", err)
	}

	s.updateRules(rules)
	return nil
}

func (s *RulesStore) updateRules(rules []model.Rule) {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	s.rules = rules
}

func getRulesFromConfigMap(cm *corev1.ConfigMap) ([]model.Rule, error) {
	if cm == nil {
		return nil, errors.New("configMap is nil")
	}

	rulesText, ok := cm.Data["rules"]
	if !ok {
		return nil, errors.New("configMap missing 'rules' key")
	}

	var xRules []model.XRule
	if err := yaml.Unmarshal([]byte(rulesText), &xRules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	rules := []model.Rule{}
	for _, x := range xRules {
		rule, err := x.ToRule()
		if err != nil {
			return nil, fmt.Errorf("ToRule err: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

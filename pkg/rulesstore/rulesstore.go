package rulesstore

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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
		return nil, fmt.Errorf("store.UpdateRules err: %w", err)
	}
	return store, nil
}

func (s *RulesStore) GetRules() *model.Rules {
	s.rulesMutex.Lock()
	defer s.rulesMutex.Unlock()

	return s.Rules
}

func (s *RulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	if cm == nil {
		return errors.New("configMap is nil")
	}
	newRules := model.Rules{}
	for key, text := range cm.Data {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		rule, err := getRuleValueFromText(text)
		if err != nil {
			return fmt.Errorf("invalid data in ConfigMap key %s: %w", key, err)
		}
		if err := validateRule(rule); err != nil {
			return fmt.Errorf("validateRule err: %w", err)
		}
		newRules[key] = *rule
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

func validateRule(rule *model.Rule) error {
	if ok := validate(rule.Namespace); !ok {
		return fmt.Errorf("invalid namespace pattern: %s", rule.Namespace)
	}
	if ok := validate(rule.Ingress); !ok {
		return fmt.Errorf("invalid ingress pattern: %s", rule.Ingress)
	}
	return nil
}

func validate(pattern string) bool {
	if pattern == "" {
		return true
	}
	regexPattern := `^!?(?:[a-z0-9\-\*]+(?:,[a-z0-9\-\*]+)*)$`
	regex := regexp.MustCompile(regexPattern)
	return regex.MatchString(pattern)
}

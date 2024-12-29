package matcher

import (
	"path/filepath"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

type Matcher struct {
	rulesStore rulesstore.IRulesStore
}

func New(rulesStore rulesstore.IRulesStore) *Matcher {
	return &Matcher{rulesStore: rulesStore}
}

func (m *Matcher) GetAnnotationsForIngress(ingress *networkingv1.Ingress) model.Annotations {
	rules := m.rulesStore.GetRules()
	result := model.Annotations{}
	for _, rule := range rules {
		if matchSelector(rule.Selector, ingress) {
			for key, value := range rule.Annotations {
				result[key] = value
			}
		}
	}
	return result
}

func matchSelector(selector model.Selector, ingress *networkingv1.Ingress) bool {
	// Check Include
	included := false
	includePatterns := strings.Split(selector.Include, ",")
	for _, pattern := range includePatterns {
		if matchPattern(pattern, ingress) {
			included = true
			break
		}
	}
	if !included {
		return false
	}

	// Check Exclude
	excludePatterns := strings.Split(selector.Exclude, ",")
	for _, pattern := range excludePatterns {
		if matchPattern(pattern, ingress) {
			return false
		}
	}

	return true
}

func matchPattern(pattern string, ingress *networkingv1.Ingress) bool {
	parts := strings.Split(pattern, "/")
	if len(parts) == 2 {
		namespacePattern, namePattern := parts[0], parts[1]
		return matchString(namespacePattern, ingress.Namespace) && matchString(namePattern, ingress.Name)
	} else if len(parts) == 1 {
		namespacePattern := parts[0]
		return matchString(namespacePattern, ingress.Namespace)
	}
	return false
}

func matchString(pattern, str string) bool {
	matched, err := filepath.Match(pattern, str)
	if err != nil {
		return false
	}
	return matched
}

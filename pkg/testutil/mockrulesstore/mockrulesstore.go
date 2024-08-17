package mockrulesstore

import (
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
)

type RulesStore struct {
	mock.Mock
	Rules *model.Rules
}

func (m *RulesStore) GetRules() *model.Rules {
	args := m.Called()
	return args.Get(0).(*model.Rules)
}

func (m *RulesStore) UpdateRules(cm *corev1.ConfigMap) error {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

func New() rulesstore.IRulesStore {
	return new(RulesStore)
}

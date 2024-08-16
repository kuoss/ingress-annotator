package mockrulesstore

import (
	"github.com/stretchr/testify/mock"

	"github.com/kuoss/ingress-annotator/controller/model"
	"github.com/kuoss/ingress-annotator/controller/rulesstore"
)

type RulesStore struct {
	mock.Mock
	Rules *model.Rules
}

func (m *RulesStore) GetRules() *model.Rules {
	args := m.Called()
	return args.Get(0).(*model.Rules)
}

func (m *RulesStore) UpdateRules() error {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(error)
}

func New() rulesstore.IRulesStore {
	return new(RulesStore)
}

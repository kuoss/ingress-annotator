package mockrulesstore_test

import (
	"errors"
	"testing"

	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/testutil/mockrulesstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetRules(t *testing.T) {
	sampleRules := &model.Rules{}
	mockStore := new(mockrulesstore.RulesStore)
	mockStore.On("GetRules").Return(sampleRules)

	result := mockStore.GetRules()
	assert.Equal(t, sampleRules, result)
}

func TestUpdateRulesSuccess(t *testing.T) {
	mockStore := new(mockrulesstore.RulesStore)
	mockStore.On("UpdateRules").Return(nil)

	err := mockStore.UpdateRules(nil)
	assert.NoError(t, err)
}

func TestUpdateRulesError(t *testing.T) {
	mockStore := new(mockrulesstore.RulesStore)
	expectedError := errors.New("update failed")
	mockStore.On("UpdateRules").Return(expectedError)

	err := mockStore.UpdateRules(nil)

	assert.EqualError(t, err, "update failed")
	mockStore.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	want := &mockrulesstore.RulesStore{
		Mock:  mock.Mock{},
		Rules: (*model.Rules)(nil),
	}
	got := mockrulesstore.New()
	assert.Equal(t, want, got)
}

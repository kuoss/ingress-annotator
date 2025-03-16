package matcher

import (
	"testing"

	"github.com/kuoss/common/tester"
	"github.com/kuoss/ingress-annotator/pkg/model"
	"github.com/kuoss/ingress-annotator/pkg/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAnnotationsForIngress(t *testing.T) {
	rules := []model.Rule{
		{
			Selector: model.Selector{
				Include: "default/*",
			},
			Annotations: map[string]string{
				"key1": "value1",
			},
		},
		{
			Selector: model.Selector{
				Include: "default/*",
				Exclude: "default/exclude-ingress",
			},
			Annotations: map[string]string{
				"key2": "value2",
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	rulesStore := mocks.NewMockIRulesStore(mockCtrl)
	rulesStore.EXPECT().GetRules().Return(rules).AnyTimes()
	matcher := New(rulesStore)

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-ingress",
		},
	}

	excludeIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "exclude-ingress",
		},
	}

	t.Run("Matching Ingress", func(t *testing.T) {
		annotations := matcher.GetAnnotationsForIngress(ingress)

		if len(annotations) != 2 {
			t.Errorf("expected 2 annotations, got %d", len(annotations))
		}

		if annotations["key1"] != "value1" {
			t.Errorf("expected annotation key1 to be value1, got %s", annotations["key1"])
		}

		if annotations["key2"] != "value2" {
			t.Errorf("expected annotation key2 to be value2, got %s", annotations["key2"])
		}
	})

	t.Run("Excluded Ingress", func(t *testing.T) {
		annotations := matcher.GetAnnotationsForIngress(excludeIngress)

		if len(annotations) != 1 {
			t.Errorf("expected 1 annotation, got %d", len(annotations))
		}

		if annotations["key1"] != "value1" {
			t.Errorf("expected annotation key1 to be value1, got %s", annotations["key1"])
		}

		if _, exists := annotations["key2"]; exists {
			t.Errorf("expected key2 to be excluded, but it exists")
		}
	})
}

func TestMatchSelector(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	rulesStore := mocks.NewMockIRulesStore(mockCtrl)
	rules := []model.Rule{{
		Description: "rule1",
		Annotations: model.Annotations{"new-key": "new-value"},
	}}
	rulesStore.EXPECT().GetRules().Return(rules).AnyTimes()

	tests := []struct {
		name     string
		selector model.Selector
		ingress  *networkingv1.Ingress
		expected bool
	}{
		{
			name: "Include matches",
			selector: model.Selector{
				Include: "namespace1/ingress1",
			},
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace1",
					Name:      "ingress1",
				},
			},
			expected: true,
		},
		{
			name: "Include does not match",
			selector: model.Selector{
				Include: "namespace1/ingress2",
			},
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace1",
					Name:      "ingress1",
				},
			},
			expected: false,
		},
		{
			name: "Exclude matches",
			selector: model.Selector{
				Exclude: "namespace1/ingress1",
			},
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace1",
					Name:      "ingress1",
				},
			},
			expected: false,
		},
		{
			name: "Exclude does not match",
			selector: model.Selector{
				Exclude: "namespace1/ingress2",
			},
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace1",
					Name:      "ingress1",
				},
			},
			expected: false,
		},
		{
			name: "Include and Exclude",
			selector: model.Selector{
				Include: "namespace1/*",
				Exclude: "namespace1/ingress1",
			},
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "namespace1",
					Name:      "ingress2",
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := matchSelector(test.selector, test.ingress)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		ingress  *networkingv1.Ingress
		expected bool
	}{
		{
			name:    "Match namespace and name",
			pattern: "test-namespace/test-name",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			expected: true,
		},
		{
			name:    "Match only namespace",
			pattern: "test-namespace",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "another-name",
				},
			},
			expected: true,
		},
		{
			name:    "No match with namespace",
			pattern: "wrong-namespace",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			expected: false,
		},
		{
			name:    "No match with namespace and name",
			pattern: "test-namespace/wrong-name",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			expected: false,
		},
		{
			name:    "Wildcard match",
			pattern: "test-namespace/*",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "any-name",
				},
			},
			expected: true,
		},
		{
			name:    "Wildcard match namespace",
			pattern: "*/test-name",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "any-namespace",
					Name:      "test-name",
				},
			},
			expected: true,
		},
		{
			name:    "Invalid pattern",
			pattern: "test-namespace/[",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			expected: false,
		},
		{
			name:    "Invalid pattern",
			pattern: "//",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-name",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.ingress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchString(t *testing.T) {
	tests := []struct {
		pattern  string
		str      string
		expected bool
	}{
		{"*", "test", true},
		{"test", "test", true},
		{"test", "different", false},
		{"prefix-*", "prefix-test", true},
		{"prefix-*", "different", false},
		{"*-suffix", "test-suffix", true},
		{"*-suffix", "test-prefix", false},
		{"*/test", "namespace/test", true},
		{"*/test", "namespace/different", false},
	}

	for _, tt := range tests {
		t.Run(tester.CaseName(tt), func(t *testing.T) {
			got := matchString(tt.pattern, tt.str)
			assert.Equal(t, tt.expected, got)
		})
	}
}

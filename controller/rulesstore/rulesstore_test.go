package rulesstore

import (
	"testing"

	"github.com/jmnote/tester"
	"github.com/kuoss/ingress-annotator/controller/fakeclient"
	"github.com/kuoss/ingress-annotator/controller/model"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		objects    []client.Object
		clientOpts *fakeclient.ClientOpts
		want       *RulesStore
		wantError  string
	}{
		{
			objects:   nil,
			wantError: "store.UpdateRules err: ConfigMap default/ingress-annotator not found",
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
				},
			},
			want: &RulesStore{
				nn:    types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
				Rules: &model.Rules{},
			},
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
				},
			},
			clientOpts: &fakeclient.ClientOpts{GetError: true},
			wantError:  "store.UpdateRules err: mocked Get error",
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": "",
					},
				},
			},
			want: &RulesStore{
				nn:    types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
				Rules: &model.Rules{},
			},
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
					},
				},
			},
			want: &RulesStore{
				nn: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
				Rules: &model.Rules{
					"rule1": model.Rule{
						Annotations: model.Annotations{"key1": "value1"},
						Namespace:   "test-namespace",
						Ingress:     "test-ingress",
					}},
			},
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
					},
				},
			},
			want: &RulesStore{
				nn: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
				Rules: &model.Rules{
					"rule1": model.Rule{
						Annotations: model.Annotations{"key1": "value1"},
						Namespace:   "test-namespace",
						Ingress:     "test-ingress",
					}},
			},
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `invalid_data`,
					},
				},
			},
			wantError: "store.UpdateRules err: invalid data in ConfigMap key rule1: failed to unmarshal YAML: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rule",
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `annotations:
  key1: value1
namespace: test-namespace
ingress: test-ingress`,
					},
				},
			},
			want: &RulesStore{
				nn: types.NamespacedName{Namespace: "default", Name: "ingress-annotator"},
				Rules: &model.Rules{
					"rule1": model.Rule{
						Annotations: model.Annotations{"key1": "value1"},
						Namespace:   "test-namespace",
						Ingress:     "test-ingress",
					}},
			},
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `invalid_data`,
					},
				},
			},
			wantError: "store.UpdateRules err: invalid data in ConfigMap key rule1: failed to unmarshal YAML: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into model.Rule",
		},
		{
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-annotator",
						Namespace: "default",
					},
					Data: map[string]string{
						"rule1": `namespace: invalid_namespace!`,
					},
				},
			},
			wantError: "store.UpdateRules err: validateRule err: invalid namespace pattern: invalid_namespace!",
		},
	}
	for i, tc := range testCases {
		t.Run(tester.Name(i, tc), func(t *testing.T) {
			nn := types.NamespacedName{Namespace: "default", Name: "ingress-annotator"}
			client := fakeclient.NewClient(tc.clientOpts, tc.objects...)
			got, err := New(client, nn)
			if tc.wantError == "" {
				assert.NoError(t, err)
				tc.want.client = got.client
				tc.want.rulesMutex = got.rulesMutex
				assert.Equal(t, tc.want, got)
				assert.Equal(t, tc.want.Rules, got.GetRules())
			} else {
				assert.EqualError(t, err, tc.wantError)
				assert.Equal(t, got, tc.want)
			}
		})
	}
}

// func TestRulesStore(t *testing.T) {
// 	mockData := map[string]string{
// 		"rule1": `
// annotations:
//   key1: value1
// namespace: test-namespace
// ingress: test-ingress`,
// 		"rule2": `
// annotations:
//   key2: value2
// namespace: another-namespace`,
// 	}
// 	cm := &corev1.ConfigMap{
// 		Data: mockData,
// 	}

// 	store := New()
// 	err := store.UpdateRules(cm)
// 	assert.NoError(t, err)

// 	want := &model.Rules{
// 		"rule1": {
// 			Annotations: model.Annotations{"key1": "value1"},
// 			Namespace:   "test-namespace",
// 			Ingress:     "test-ingress",
// 		},
// 		"rule2": {
// 			Annotations: model.Annotations{"key2": "value2"},
// 			Namespace:   "another-namespace",
// 			Ingress:     "",
// 		},
// 	}
// 	got := store.GetRules()
// 	assert.Equal(t, want, got)
// }

// func TestGetRuleValueFromText(t *testing.T) {
// 	yamlText := `
// annotations:
//   key1: value1
// namespace: test-namespace
// ingress: test-ingress`
// 	want := &model.Rule{
// 		Annotations: model.Annotations{"key1": "value1"},
// 		Namespace:   "test-namespace",
// 		Ingress:     "test-ingress",
// 	}

// 	got, err := getRuleValueFromText(yamlText)
// 	assert.NoError(t, err)
// 	assert.Equal(t, want, got)
// }

// func TestConcurrency(t *testing.T) {
// 	store := New()
// 	store.Rules = &model.Rules{
// 		"initialRule": {
// 			Annotations: model.Annotations{"initialKey": "initialValue"},
// 			Namespace:   "initial-namespace",
// 			Ingress:     "initial-ingress",
// 		},
// 	}

// 	var wg sync.WaitGroup

// 	// Test concurrent access to GetRules and UpdateRules
// 	for i := 0; i < 100; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			store.GetRules()
// 		}()

// 		wg.Add(1)
// 		go func(_ int) {
// 			defer wg.Done()
// 			mockData := map[string]string{
// 				"rule": `
// annotations:
//   key: value
// namespace: namespace`,
// 			}
// 			cm := &corev1.ConfigMap{
// 				Data: mockData,
// 			}
// 			_ = store.UpdateRules(cm)
// 		}(i)
// 	}

// 	wg.Wait()
// }

// func TestUpdateRules(t *testing.T) {
// 	testCases := []struct {
// 		name          string
// 		configMapData map[string]string
// 		wantError     string
// 	}{
// 		{
// 			name: "Valid rule",
// 			configMapData: map[string]string{
// 				"rule1": "namespace: test-namespace\ningress: test-ingress",
// 			},
// 			wantError: "",
// 		},
// 		{
// 			name: "Invalid namespace pattern",
// 			configMapData: map[string]string{
// 				"rule1": "namespace: invalid_namespace!\ningress: test-ingress",
// 			},
// 			wantError: "store.UpdateRules err: validateRule err: invalid namespace pattern: invalid_namespace!",
// 		},
// 		{
// 			name: "Invalid ingress pattern",
// 			configMapData: map[string]string{
// 				"rule1": "namespace: test-namespace\ningress: invalid_ingress!",
// 			},
// 			wantError: "",
// 		},
// 		{
// 			name: "Invalid YAML",
// 			configMapData: map[string]string{
// 				"rule1": "namespace: test-namespace\ningress",
// 			},
// 			wantError: "",
// 		},
// 		{
// 			name:          "Empty ConfigMap",
// 			configMapData: map[string]string{},
// 			wantError:     "",
// 		},
// 	}

// 	for i, tc := range testCases {
// 		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
// 			t.Setenv("POD_NAMESPACE", "default")
// 			cm := &corev1.ConfigMap{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "ingress-annotator",
// 					Namespace: "default",
// 				},
// 				Data: tc.configMapData,
// 			}
// 			client := testutil.NewFakeClient(nil, cm)
// 			store, err := New(client)
// 			assert.NoError(t, err)

// 			err = store.UpdateRules()
// 			if tc.wantError == "" {
// 				assert.NoError(t, err)
// 			} else {
// 				assert.EqualError(t, err, tc.wantError)
// 			}
// 		})
// 	}
// }

func TestValidateRule(t *testing.T) {
	testCases := []struct {
		name      string
		rule      model.Rule
		wantError string
	}{
		{
			name: "Valid Rule with Namespace and Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "ingress-1",
			},
			wantError: "",
		},
		{
			name: "Valid Rule with Namespace and empty Ingress",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "",
			},
			wantError: "",
		},
		{
			name: "Invalid Namespace pattern",
			rule: model.Rule{
				Namespace: "Invalid_Namespace",
				Ingress:   "ingress-1",
			},
			wantError: "invalid namespace pattern: Invalid_Namespace",
		},
		{
			name: "Invalid Ingress pattern",
			rule: model.Rule{
				Namespace: "namespace-1",
				Ingress:   "Invalid,Ingress",
			},
			wantError: "invalid ingress pattern: Invalid,Ingress",
		},
		{
			name: "Valid Namespace with wildcard and negation",
			rule: model.Rule{
				Namespace: "namespace-*",
				Ingress:   "!ingress-*",
			},
			wantError: "",
		},
		{
			name: "Invalid pattern with special characters",
			rule: model.Rule{
				Namespace: "namespace-!",
				Ingress:   "ingress-1",
			},
			wantError: "invalid namespace pattern: namespace-!",
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.name), func(t *testing.T) {
			err := validateRule(&tc.rule)
			if tc.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantError)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		pattern string
		want    bool
	}{
		// true
		{"", true},                 // Empty string
		{"abc", true},              // Single string
		{"abc,def", true},          // Comma-separated strings
		{"!abc", true},             // Pattern with an exclamation mark
		{"!abc,def", true},         // Pattern with exclamation mark and comma-separated strings
		{"abc-def", true},          // String with a hyphen
		{"abc*", true},             // String with an asterisk
		{"abc,def,ghi", true},      // Multiple comma-separated strings
		{"!abc,def-ghi", true},     // Pattern with exclamation mark and hyphen
		{"abc,def-ghi,jkl*", true}, // Combination of comma, hyphen, and asterisk
		// false
		{"abc,", false},          // Invalid pattern ending with a comma
		{"abc,def!", false},      // Invalid pattern with an exclamation mark in the wrong place
		{"!abc,", false},         // Invalid pattern ending with a comma after exclamation mark
		{"abc,def*", true},       // Pattern with an asterisk
		{"!abc,def*", true},      // Pattern with exclamation mark and asterisk
		{"abc@def", false},       // Invalid pattern with an incorrect special character
		{"abc,def,", false},      // Invalid comma placement
		{"!abc,def,ghi!", false}, // Exclamation mark in the wrong place
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.pattern), func(t *testing.T) {
			got := validate(tc.pattern)
			assert.Equal(t, tc.want, got)
		})
	}
}

package fake

import (
	"github.com/kuoss/ingress-annotator/pkg/rulesstore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RulesStore struct {
}

func (s RulesStore) ConfigMapNamespace() string {
	return "fake-namespace"
}

func (s RulesStore) ConfigMapName() string {
	return "fake-name"
}

func (s RulesStore) GetData() *rulesstore.Data {
	return &rulesstore.Data{
		ConfigMap: corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{},
			Data:       map[string]string{},
			BinaryData: map[string][]byte{},
		},
		Rules: map[string]rulesstore.Annotations{"rules1": {"foo": "bar"}},
	}
}

func (s RulesStore) UpdateData() error {
	return nil
}

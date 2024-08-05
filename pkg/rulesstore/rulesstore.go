package rulesstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Annotations map[string]string

type Rules map[string]Annotations

type Data struct {
	ConfigMap corev1.ConfigMap
	Rules     Rules
}

type RulesStore struct {
	Client             client.Client
	ConfigMapNamespace string
	ConfigMapName      string
	Data               Data
	DataMutex          *sync.Mutex
}

func New(client client.Client) (*RulesStore, error) {
	ns, exists := os.LookupEnv("POD_NAMESPACE")
	if !exists || ns == "" {
		return nil, errors.New("POD_NAMESPACE environment variable is not set or is empty")
	}

	rulesStore := &RulesStore{
		Client:             client,
		ConfigMapNamespace: ns,
		ConfigMapName:      configMapName,
		Data:               Data{},
		DataMutex:          &sync.Mutex{},
	}

	if err := rulesStore.UpdateData(); err != nil {
		return nil, fmt.Errorf("update data error: %w", err)
	}

	return rulesStore, nil
}

func (s *RulesStore) GetData() *Data {
	s.DataMutex.Lock()
	defer s.DataMutex.Unlock()

	return &s.Data
}

func (s *RulesStore) UpdateData() error {
	var cm corev1.ConfigMap
	if err := s.Client.Get(
		context.Background(),
		client.ObjectKey{Namespace: s.ConfigMapNamespace, Name: s.ConfigMapName},
		&cm,
	); err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}
	rules := Rules{}
	for key, text := range cm.Data {
		annotations, err := getAnnotationsFromText(text)
		if err != nil {
			return fmt.Errorf("invalid data in ConfigMap key %s: %w", key, err)
		}
		rules[key] = annotations
	}

	s.DataMutex.Lock()
	defer s.DataMutex.Unlock()

	s.Data = Data{
		ConfigMap: cm,
		Rules:     rules,
	}

	return nil
}

func getAnnotationsFromText(text string) (Annotations, error) {
	annotations := make(Annotations)
	err := yaml.Unmarshal([]byte(text), &annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}
	return annotations, nil
}

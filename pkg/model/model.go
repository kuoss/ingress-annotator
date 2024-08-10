package model

type Annotations map[string]string

type AnnotationsUnmanagePolicy string

const (
	AnnotationsUnmanagePolicyDelete AnnotationsUnmanagePolicy = "Delete"
	AnnotationsUnmanagePolicyRetain AnnotationsUnmanagePolicy = "Retain"
)

type Rules map[string]Rule

type Rule struct {
	Annotations Annotations `yaml:"annotations"`
	Namespace   string      `yaml:"namespace"`
	Ingress     string      `yaml:"ingress,omitempty"`
}

package model

import (
	"errors"
	"strings"
)

type Annotations map[string]string

type Selector struct {
	Include string `yaml:"include,omitempty"`
	Exclude string `yaml:"exclude,omitempty"`
}

type Rule struct {
	Description string      `yaml:"description,omitempty"`
	Selector    Selector    `yaml:"selector"`
	Annotations Annotations `yaml:"annotations,omitempty"`
}

type ListAnnotations map[string][]string

type XRule struct {
	Description     string          `yaml:"description,omitempty"`
	Selector        Selector        `yaml:"selector"`
	Annotations     Annotations     `yaml:"annotations,omitempty"`
	ListAnnotations ListAnnotations `yaml:"listAnnotations,omitempty"`
}

// ToRule converts XRule to Rule by merging ListAnnotations into Annotations
func (x XRule) ToRule() (Rule, error) {
	mergedAnnotations := make(Annotations)

	// Copy existing annotations
	for key, value := range x.Annotations {
		mergedAnnotations[key] = value
	}

	// Check for key conflicts before merging ListAnnotations
	for key := range x.ListAnnotations {
		if _, exists := mergedAnnotations[key]; exists {
			return Rule{}, errors.New("conflicting key found in Annotations and ListAnnotations: " + key)
		}
	}

	// Convert ListAnnotations to comma-separated strings
	for key, values := range x.ListAnnotations {
		mergedAnnotations[key] = strings.Join(values, ",")
	}

	return Rule{
		Description: x.Description,
		Selector:    x.Selector,
		Annotations: mergedAnnotations,
	}, nil
}

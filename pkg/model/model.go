package model

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

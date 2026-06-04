package dsl

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadDiagram reads a YAML file and parses it into a Diagram.
func LoadDiagram(path string) (*Diagram, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDiagram(data)
}

// ParseDiagram parses YAML bytes into a Diagram.
func ParseDiagram(data []byte) (*Diagram, error) {
	d := &Diagram{}
	if err := yaml.Unmarshal(data, d); err != nil {
		return nil, err
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return d, nil
}

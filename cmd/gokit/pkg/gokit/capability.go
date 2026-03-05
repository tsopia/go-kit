package gokit

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed capabilities.yaml
var capabilitiesFS embed.FS

type Scenario struct {
	Name    string `yaml:"name"`
	Snippet string `yaml:"snippet"`
}

type Capability struct {
	Name         string     `yaml:"name"`
	Description  string     `yaml:"description"`
	Import       string     `yaml:"import"`
	Scenarios    []Scenario `yaml:"scenarios"`
	Dependencies []string   `yaml:"dependencies"`
}

type CapabilityRegistry struct {
	Version      string       `yaml:"version"`
	UpdatedAt    string       `yaml:"updated_at"`
	Capabilities []Capability `yaml:"capabilities"`
}

func LoadCapabilities() ([]Capability, error) {
	data, err := capabilitiesFS.ReadFile("capabilities.yaml")
	if err != nil {
		return nil, fmt.Errorf("read capabilities: %w", err)
	}

	var registry CapabilityRegistry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse capabilities: %w", err)
	}

	return registry.Capabilities, nil
}

func GetCapability(name string) (*Capability, error) {
	caps, err := LoadCapabilities()
	if err != nil {
		return nil, err
	}

	for _, c := range caps {
		if c.Name == name {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("capability not found: %s", name)
}

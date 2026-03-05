package gokit

import (
	"embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed capabilities.yaml
var capabilitiesFS embed.FS

const CapabilitiesPath = ".ai/capabilities.yaml"

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

// LoadCapabilities 从项目根目录读取，如果不存在则使用 embedded
func LoadCapabilities() ([]Capability, error) {
	// 先尝试从项目根目录读取
	if data, err := os.ReadFile(CapabilitiesPath); err == nil {
		return parseCapabilities(data)
	}

	// 使用 embedded 版本
	data, err := capabilitiesFS.ReadFile("capabilities.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded capabilities: %w", err)
	}

	return parseCapabilities(data)
}

func parseCapabilities(data []byte) ([]Capability, error) {
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

package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the CV configuration.
type Config struct {
	Langs  []Lang  `yaml:"langs"`
	Layers []Layer `yaml:"layers"`
}

// Lang defines an available language.
type Lang struct {
	Code  string `yaml:"code"`
	Label string `yaml:"label"`
}

// Layer defines a key-driven command.
type Layer struct {
	Key   string            `yaml:"key"`
	Cmd   map[string]string `yaml:"cmd"`
	Title map[string]string `yaml:"title"`
	Label map[string]string `yaml:"label"`
}

// Load reads and validates a YAML config file.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if len(cfg.Layers) == 0 {
		return cfg, fmt.Errorf("aucun layer defini")
	}
	if len(cfg.Langs) == 0 {
		return cfg, fmt.Errorf("aucune langue definie")
	}
	for index, lang := range cfg.Langs {
		if strings.TrimSpace(lang.Code) == "" {
			return cfg, fmt.Errorf("langue sans code a l'index %d", index)
		}
	}

	return cfg, nil
}

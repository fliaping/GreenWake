package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`

	HTTP struct {
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"http"`

	Hosts []struct {
		Name        string `yaml:"name"`
		IP          string `yaml:"ip"`
		MAC         string `yaml:"mac"`
		MonitorPort int    `yaml:"monitor_port"`
	} `yaml:"hosts"`

	Forwards []struct {
		ServicePort int    `yaml:"service_port"`
		TargetHost  string `yaml:"target_host"`
		TargetPort  int    `yaml:"target_port"`
	} `yaml:"forwards"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

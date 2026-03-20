package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Server string `yaml:"server"`
	APIKey string `yaml:"api_key"`
}

type Config struct {
	Services   []ServiceConfig `yaml:"services"`
	DefaultSvc int             `yaml:"default_service"` // 1-based index
	Paths      []PathConfig    `yaml:"paths"`
}

type PathConfig struct {
	Name    string `yaml:"name"`
	Port    int    `yaml:"port"`
	Service int    `yaml:"service"` // Index to Services
}

func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "l2h", "client.yaml")
}

func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

package main

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Timeout        int    `yaml:"timeout"` // minutes
	StartCommand   string `yaml:"start_command"`
	Port           string `yaml:"port"`
	ConnectTimeout int    `yaml:"connect_timeout"` //seconds
}

// this is a global constant since it's shared
var config Config

func LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		GetLogger().Fatalf("Failed to read config file: %v", err)
		return err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		GetLogger().Fatalf("Failed to unmarshal config: %v", err)
		return err
	}
	GetLogger().Info("Configuration loaded successfully")
	return nil
}

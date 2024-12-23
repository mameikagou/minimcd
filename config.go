package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

// 配置结构体
type Config struct {
	Server struct {
		Timeout      int    `yaml:"timeout"`
		StartCommand string `yaml:"start_command"`
		Address      string `yaml:"address"`
	} `yaml:"server"`
}

// 全局配置变量
var config Config

// 加载配置文件
func loadConfig(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		getLogger().Errorf("Failed to read config file: %v", err)
		return err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		getLogger().Errorf("Failed to unmarshal config: %v", err)
		return err
	}
	getLogger().Info("Configuration loaded successfully")
	return nil
}

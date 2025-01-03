package config

import (
	"awake/pkg/logger"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type WolWakeConfig struct {
	WolPort           int `yaml:"wol_port"`
	WolTimeoutMinutes int `yaml:"wol_timeout_minutes"`
}

type Config struct {
	Strategy  string        `yaml:"strategy"`
	SleepMode string        `yaml:"sleep_mode"`
	WolWake   WolWakeConfig `yaml:"wol_wake"`
}

func GetConfigPath() string {
	var configDir string
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "awake")
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "awake")
	case "linux":
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "awake")
	default:
		configDir = "."
	}
	return filepath.Join(configDir, "config.yaml")
}

func LoadConfig(configPath string) (*Config, error) {
	// 如果没有指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = GetConfigPath()
	}

	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置
		cfg := &Config{
			Strategy:  "wol_wake",
			SleepMode: "system",
			WolWake: WolWakeConfig{
				WolPort:           9,
				WolTimeoutMinutes: 10,
			},
		}
		if err := SaveConfig(cfg, configPath); err != nil {
			return nil, err
		}
		logger.Info("创建新配置文件: %s", configPath)
		logger.Info("配置内容: %+v", cfg)
		return cfg, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	logger.Info("加载现有配置文件: %s", configPath)
	logger.Info("配置内容: %+v", cfg)
	return &cfg, nil
}

func SaveConfig(cfg *Config, configPath string) error {
	if configPath == "" {
		configPath = GetConfigPath()
	}

	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	return encoder.Encode(cfg)
}

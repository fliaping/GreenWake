package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// 默认配置值
	DefaultLogLevel        = "info" // 默认日志级别
	DefaultHTTPPort        = "8055" // 默认HTTP端口
	DefaultRefreshInterval = 30     // 默认刷新间隔（秒）
	DefaultWakeTimeout     = 10     // 默认唤醒超时时间（秒）
	DefaultRetryCount      = 1      // 默认重试次数
	DefaultWakeInterval    = 5      // 默认唤醒间隔（秒）
)

type PCHostConfig struct {
	Name         string `yaml:"name"`
	IP           string `yaml:"ip"`
	MAC          string `yaml:"mac"`
	MonitorPort  int    `yaml:"monitor_port"`
	WakeTimeout  int    `yaml:"wake_timeout"`
	RetryCount   int    `yaml:"retry_count"`
	WakeInterval int    `yaml:"wake_interval"`
}

type Config struct {
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`

	HTTP struct {
		Port            string `yaml:"port"`
		User            string `yaml:"user"`
		Password        string `yaml:"password"`
		RefreshInterval int    `yaml:"refresh_interval"`
	} `yaml:"http"`

	Hosts []PCHostConfig `yaml:"hosts"`

	Forwards []struct {
		ServicePort int    `yaml:"service_port"`
		TargetHost  string `yaml:"target_host"`
		TargetPort  int    `yaml:"target_port"`
	} `yaml:"forwards"`
}

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
		}
		exeDir := filepath.Dir(exePath)

		examplePaths := []string{
			filepath.Join(exeDir, "config.example.yaml"),
			filepath.Join(exeDir, "config.example.yml"),
			"config.example.yaml",
			"config.example.yml",
		}

		var exampleConfig []byte
		var foundExample bool
		for _, examplePath := range examplePaths {
			if data, err := os.ReadFile(examplePath); err == nil {
				exampleConfig = data
				foundExample = true
				break
			}
		}

		if !foundExample {
			cfg := &Config{
				Log: struct {
					Level string `yaml:"level"`
				}{
					Level: DefaultLogLevel,
				},
				HTTP: struct {
					Port            string `yaml:"port"`
					User            string `yaml:"user"`
					Password        string `yaml:"password"`
					RefreshInterval int    `yaml:"refresh_interval"`
				}{
					Port:            DefaultHTTPPort,
					RefreshInterval: DefaultRefreshInterval,
				},
			}

			configDir := filepath.Dir(path)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("创建配置目录失败: %v", err)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return nil, fmt.Errorf("序列化默认配置失败: %v", err)
			}

			if err := os.WriteFile(path, data, 0644); err != nil {
				return nil, fmt.Errorf("写入默认配置失败: %v", err)
			}

			return cfg, nil
		}

		configDir := filepath.Dir(path)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("创建配置目录失败: %v", err)
		}

		if err := os.WriteFile(path, exampleConfig, 0644); err != nil {
			return nil, fmt.Errorf("复制示例配置失败: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.Log.Level == "" {
		cfg.Log.Level = DefaultLogLevel
	}
	if cfg.HTTP.Port == "" {
		cfg.HTTP.Port = DefaultHTTPPort
	}
	if cfg.HTTP.RefreshInterval == 0 {
		cfg.HTTP.RefreshInterval = DefaultRefreshInterval
	}
	// 设置主机配置的默认值
	for i := range cfg.Hosts {
		if cfg.Hosts[i].WakeTimeout == 0 {
			cfg.Hosts[i].WakeTimeout = DefaultWakeTimeout
		}
		if cfg.Hosts[i].RetryCount == 0 {
			cfg.Hosts[i].RetryCount = DefaultRetryCount
		}
		if cfg.Hosts[i].WakeInterval == 0 {
			cfg.Hosts[i].WakeInterval = DefaultWakeInterval
		}
	}

	return &cfg, nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "config.yaml"
	}

	// 拼接应用配置目录
	appConfigDir := filepath.Join(configDir, "greenwake-bridge")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "config.yaml"
	}

	return filepath.Join(appConfigDir, "config.yaml")
}

package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	// 程序控制睡眠模式下等待睡眠时间（秒）
	ProgramSleepDelay int `yaml:"program_sleep_delay"`

	// 唤醒包相关配置
	WolWake WolWakeConfig `yaml:"wol_wake"`

	// 自动启动配置
	AutoStart bool `yaml:"auto_start"`

	// 初始配置
	Initial struct {
		Strategy  string `yaml:"strategy"`   // 初始唤醒策略
		SleepMode string `yaml:"sleep_mode"` // 初始睡眠模式
		Duration  string `yaml:"duration"`   // 初始计时时长
	} `yaml:"initial"`
}

// WolWakeConfig WOL唤醒配置
type WolWakeConfig struct {
	WolPort        int    `yaml:"wol_port"`         // WOL端口
	WolTimeoutSecs int    `yaml:"wol_timeout_secs"` // WOL超时时间（秒）
	ValidEvents    string `yaml:"valid_events"`     // 有效的唤醒事件类型，多个类型用逗号分隔
}

// GetValidEvents 获取有效的唤醒事件类型列表
func (c *WolWakeConfig) GetValidEvents() []string {
	if c.ValidEvents == "" {
		return []string{"wol", "keyboard", "mouse"} // 默认值
	}
	events := strings.Split(c.ValidEvents, ",")
	// 去除空格
	for i := range events {
		events[i] = strings.TrimSpace(events[i])
	}
	return events
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
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
	appConfigDir := filepath.Join(configDir, "awake")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "config.yaml"
	}

	return filepath.Join(appConfigDir, "config.yaml")
}

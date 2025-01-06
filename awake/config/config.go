package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// 默认配置值
	DefaultStrategy          = "external_wake"
	DefaultSleepMode         = "program"
	DefaultTimedDuration     = "30m"
	DefaultProgramSleepDelay = 60
	DefaultWolPort           = 9
	DefaultTimeoutSecs       = 300
	DefaultValidEvents       = "wol,device"
	DefaultLogLevel          = "debug" // 默认日志级别
)

// Config 配置结构
type Config struct {
	Strategy          string       `yaml:"strategy"`            // 唤醒策略
	SleepMode         string       `yaml:"sleep_mode"`          // 睡眠模式
	TimedDuration     string       `yaml:"timed_duration"`      // 定时唤醒持续时间
	ExternalWake      ExternalWake `yaml:"external_wake"`       // 外部唤醒相关配置
	ProgramSleepDelay int          `yaml:"program_sleep_delay"` // 程序控制睡眠模式下等待睡眠时间
	LogLevel          string       `yaml:"log_level"`           // 日志级别
}

// ExternalWake 外部唤醒相关配置
type ExternalWake struct {
	WolPort     int    `yaml:"wol_port"`     // 唤醒包监听端口
	TimeoutSecs int    `yaml:"timeout_secs"` // 唤醒超时时间
	ValidEvents string `yaml:"valid_events"` // 有效的唤醒事件类型
}

// GetValidEvents 获取有效的唤醒事件类型列表
func (w *ExternalWake) GetValidEvents() []string {
	if w.ValidEvents == "" {
		return strings.Split(DefaultValidEvents, ",")
	}
	return strings.Split(w.ValidEvents, ",")
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

	// 设置默认值
	if cfg.Strategy == "" {
		cfg.Strategy = DefaultStrategy
	}
	if cfg.SleepMode == "" {
		cfg.SleepMode = DefaultSleepMode
	}
	if cfg.TimedDuration == "" {
		cfg.TimedDuration = DefaultTimedDuration
	}
	if cfg.ProgramSleepDelay == 0 {
		cfg.ProgramSleepDelay = DefaultProgramSleepDelay
	}
	if cfg.ExternalWake.WolPort == 0 {
		cfg.ExternalWake.WolPort = DefaultWolPort
	}
	if cfg.ExternalWake.TimeoutSecs == 0 {
		cfg.ExternalWake.TimeoutSecs = DefaultTimeoutSecs
	}
	if cfg.ExternalWake.ValidEvents == "" {
		cfg.ExternalWake.ValidEvents = DefaultValidEvents
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = DefaultLogLevel
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

// IsEventTypeValid 检查事件类型是否有效
func (c *Config) IsEventTypeValid(eventType string) bool {
	validEvents := c.ExternalWake.GetValidEvents()
	for _, event := range validEvents {
		if event == eventType {
			return true
		}
	}
	return false
}
